#!/bin/sh
# Veckomenyn one-command remote install via Tailscale.
#
# Drops the upstream docker-compose.yml + docker-compose.tailscale.yml
# into ~/veckomenyn (override with $VECKOMENYN_DIR), writes a tiny .env
# with your Tailscale auth key, and starts the stack. The app is only
# reachable via your tailnet. No public ports, no DNS, no TLS to
# manage.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/simonnordberg/veckomenyn/main/install.sh \
#     | TS_AUTHKEY=tskey-... sh
#
# Or interactive (will prompt for the auth key):
#   curl -fsSL https://raw.githubusercontent.com/simonnordberg/veckomenyn/main/install.sh | sh
#
# Get an auth key at https://login.tailscale.com/admin/settings/keys

set -eu

REPO_RAW="https://raw.githubusercontent.com/simonnordberg/veckomenyn/main"
INSTALL_DIR="${VECKOMENYN_DIR:-$HOME/veckomenyn}"

err() { printf '%s\n' "$*" >&2; }

# --- detect a compose engine ----------------------------------------------
# Prefer docker. On a cloud VM rootful Docker is the smoother path: its
# systemd unit is enabled at install, so containers come back at boot
# without extra setup. Fall back to podman if that's what's already there
# (homelab Fedora hosts, opinionated users). Either works; the rest of the
# script uses $COMPOSE so we can address whichever one is present.
if command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then
  COMPOSE="docker compose"
elif command -v podman >/dev/null 2>&1 && podman compose version >/dev/null 2>&1; then
  COMPOSE="podman compose"
else
  err "Neither docker nor podman (with the compose plugin) is installed."
  err ""

  distro="other"
  if [ -r /etc/os-release ]; then
    . /etc/os-release
    case "${ID:-}${ID_LIKE:-}" in
      *ubuntu*|*debian*)         distro="debian" ;;
      *fedora*|*rhel*|*centos*)  distro="rhel" ;;
      *arch*)                    distro="arch" ;;
      *alpine*)                  distro="alpine" ;;
    esac
  fi

  err "Install Docker (recommended for cloud VMs):"
  case "$distro" in
    debian|rhel) err "  curl -fsSL https://get.docker.com | sh" ;;
    arch)        err "  sudo pacman -S --needed docker docker-compose" ;;
    alpine)      err "  sudo apk add docker docker-cli-compose && sudo rc-update add docker default && sudo service docker start" ;;
    *)           err "  https://docs.docker.com/engine/install/" ;;
  esac
  err ""

  err "Or install Podman:"
  case "$distro" in
    debian)  err "  sudo apt update && sudo apt install -y podman podman-compose" ;;
    rhel)    err "  sudo dnf install -y podman podman-compose" ;;
    arch)    err "  sudo pacman -S --needed podman podman-compose" ;;
    alpine)  err "  sudo apk add podman podman-compose" ;;
    *)       err "  https://podman.io/getting-started/installation" ;;
  esac
  err ""

  err "Both work; rootless Podman needs extra setup to survive reboot."
  err "See https://github.com/simonnordberg/veckomenyn/blob/main/docs/deploy-tailscale.md#surviving-reboots"
  exit 1
fi

# --- get the auth key -----------------------------------------------------
if [ -z "${TS_AUTHKEY:-}" ]; then
  if [ ! -t 0 ]; then
    err "TS_AUTHKEY is required. Either pipe with TS_AUTHKEY set:"
    err "  curl ... | TS_AUTHKEY=tskey-... sh"
    err "Or run the script directly so it can prompt:"
    err "  curl -o install.sh ... && TS_AUTHKEY=tskey-... sh install.sh"
    exit 1
  fi
  printf 'Tailscale auth key (https://login.tailscale.com/admin/settings/keys): '
  read -r TS_AUTHKEY
fi

case "$TS_AUTHKEY" in
  tskey-*) ;;
  *) err "auth key must start with tskey-"; exit 1 ;;
esac

# --- lay down the install dir --------------------------------------------
mkdir -p "$INSTALL_DIR"
cd "$INSTALL_DIR"

curl -fsSL "$REPO_RAW/docker-compose.yml" -o docker-compose.yml
curl -fsSL "$REPO_RAW/docker-compose.tailscale.yml" -o docker-compose.tailscale.yml

# Don't clobber an existing .env on re-runs (preserves any local customisation).
if [ ! -f .env ]; then
  printf 'TS_AUTHKEY=%s\n' "$TS_AUTHKEY" > .env
else
  if ! grep -q '^TS_AUTHKEY=' .env; then
    printf 'TS_AUTHKEY=%s\n' "$TS_AUTHKEY" >> .env
  fi
fi
chmod 600 .env

# --- start ---------------------------------------------------------------
$COMPOSE -f docker-compose.yml -f docker-compose.tailscale.yml up -d

cat <<HINT

Started veckomenyn in $INSTALL_DIR using "$COMPOSE".

The app is reachable from any device on your tailnet at:
  http://veckomenyn.<your-tailnet>.ts.net:8080

Find your tailnet name at https://login.tailscale.com/admin (e.g. "tail123abc.ts.net").

To upgrade later:
  cd $INSTALL_DIR
  $COMPOSE -f docker-compose.yml -f docker-compose.tailscale.yml pull
  $COMPOSE -f docker-compose.yml -f docker-compose.tailscale.yml up -d

To stop:
  cd $INSTALL_DIR
  $COMPOSE -f docker-compose.yml -f docker-compose.tailscale.yml down
HINT
