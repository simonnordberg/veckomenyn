# Deploy on a remote VM (Tailscale)

A small cloud VM with **Tailscale as the auth layer**. The app sits on the tailnet only. No public ports, no DNS, no TLS. Tailscale's identity (you, signed into your tailnet) is the access control.

## Prerequisites

- A Linux VM with **rootful Docker**. Hetzner CX22 (€4/mo, 4 GB RAM) is plenty; a Raspberry Pi 5 at home works identically. On Ubuntu, Debian, Fedora, CentOS, or RHEL, install with Docker's official one-liner:
  ```sh
  curl -fsSL https://get.docker.com | sh
  ```
  ([Other distros / manual install](https://docs.docker.com/engine/install/).) Podman works too, but Docker is the smoother choice for a cloud VM: it ships with its systemd unit enabled, so containers come back automatically after reboot. Podman's rootless mode needs a couple of one-time commands ([Surviving reboots](#surviving-reboots)).
- A free Tailscale account ([tailscale.com](https://tailscale.com)). The free plan covers 100 devices, more than enough for a household.
- A Tailscale [auth key](https://login.tailscale.com/admin/settings/keys) (see Tailscale's [auth keys docs](https://tailscale.com/kb/1085/auth-keys)). A non-ephemeral, non-reusable key is the right default for a single VM.

## One-line install

SSH into the VM:

```sh
curl -fsSL https://raw.githubusercontent.com/simonnordberg/veckomenyn/main/install.sh \
  | TS_AUTHKEY=tskey-... sh
```

The script:

1. Picks `docker` or falls back to `podman`.
2. Downloads `docker-compose.yml` and `docker-compose.tailscale.yml` into `~/veckomenyn`.
3. Writes `.env` with the auth key (mode `600`).
4. Runs `compose -f docker-compose.yml -f docker-compose.tailscale.yml up -d`.

The Tailscale sidecar joins the tailnet under hostname `veckomenyn`. The app shares its network namespace, so the host has nothing exposed.

Open the printed URL from any tailnet-connected device:

```
http://veckomenyn.<your-tailnet>.ts.net:8080
```

Find your tailnet name in the [admin panel](https://login.tailscale.com/admin); see Tailscale's [MagicDNS docs](https://tailscale.com/kb/1081/magicdns) for the naming scheme.

## Manual install

```sh
mkdir -p ~/veckomenyn && cd ~/veckomenyn

curl -O https://raw.githubusercontent.com/simonnordberg/veckomenyn/main/docker-compose.yml
curl -O https://raw.githubusercontent.com/simonnordberg/veckomenyn/main/docker-compose.tailscale.yml

echo "TS_AUTHKEY=tskey-..." > .env
chmod 600 .env

podman compose -f docker-compose.yml -f docker-compose.tailscale.yml up -d
```

## How the overlay works

`docker-compose.tailscale.yml` adds two things to the base compose:

1. A `tailscale` service running the official [`tailscale/tailscale`](https://hub.docker.com/r/tailscale/tailscale) image. Joins the tailnet using `TS_AUTHKEY`, persists state in a named volume so reboots don't re-auth. See Tailscale's [Docker recipe](https://tailscale.com/kb/1282/docker) for the full pattern.
2. `network_mode: service:tailscale` on `app`, with `ports: !reset []` to drop the host port. The app is reachable only through the tailnet.

Drop the overlay and the base compose is the LAN deployment from [Quickstart](quickstart.md). Layer it back on for the tailnet path.

## Surviving reboots

`restart: unless-stopped` in the compose file brings the containers back if they crash or if the engine restarts. What it can't do alone is bring everything back after a host reboot, because that depends on the engine itself starting at boot:

- **Docker (rootful, the default on cloud VMs).** Docker installs with its systemd unit enabled by default on Debian/Ubuntu/Fedora. After reboot, the daemon starts, then the labelled containers come up. Verify with `systemctl is-enabled docker`.
- **Podman, rootful.** Same story: `systemctl is-enabled podman` (or `podman.socket`) should report `enabled`. If not: `sudo systemctl enable --now podman.socket`.
- **Podman, rootless.** Trickier: rootless services die when the user logs out and don't auto-start at boot without a couple of one-time commands. Run once: `loginctl enable-linger $USER` so the user's services survive logout, then `systemctl --user enable podman-restart.service` so restart-policy containers come back at boot.

If you're running on a cloud VM with the default OS image and rootful Docker, you don't need to do anything; it already works.

## Updating

```sh
cd ~/veckomenyn
podman compose -f docker-compose.yml -f docker-compose.tailscale.yml pull
podman compose -f docker-compose.yml -f docker-compose.tailscale.yml up -d
```

`:0.3` pins to the patch channel. See [Upgrading](upgrading.md) for other cadences and the in-app update banner. Pre-migration `pg_dump` runs on every restart with pending migrations.

## Off-host backups

[Pre-migration snapshots](backups.md) in `~/veckomenyn/backups/` survive `compose down -v` but not the VM itself. Sync the directory off the VM nightly:

```sh
# crontab
0 4 * * * restic -r b2:bucket:/veckomenyn backup ~/veckomenyn/backups
```

`restic`, `rclone`, `borg`, `rsync`. Whichever you already use. The directory is plain dump files.

## Sharing access with family

Add each family member as a [user on your tailnet](https://tailscale.com/kb/1018/install) and they install the Tailscale app on their phone or laptop ([app downloads](https://tailscale.com/download)). Once signed in, the app URL works from any of their devices.

To restrict who on the tailnet can reach veckomenyn, use [Tailscale ACLs](https://tailscale.com/kb/1018/acls). The default policy lets every tailnet user reach every node, which is fine for a single-household setup.

## Tradeoffs

- **One host, no failover.** If the VM dies, restore from off-host backups onto a new one. Veckomenyn is a household app, not a service. High-availability is the wrong shape for it.
- **Tailscale is a managed dependency.** [Headscale](https://github.com/juanfont/headscale) is the self-hosted alternative with the same compose pattern.
- **Every family device needs Tailscale.** Free apps, ~5 min per device.

[SECURITY.md](../SECURITY.md) for the threat model and on-disk encryption details.
