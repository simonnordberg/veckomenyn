# Deploy on a remote VM (Tailscale)

For accessing Veckomenyn from outside your home — phones on cellular, a laptop at a coffee shop, family members visiting elsewhere — you need a host that's reachable to you but not to the public internet. The recommended path is a small cloud VM with **Tailscale as the auth layer**: you join the tailnet from your devices, and the app sits on the tailnet only. No public ports, no DNS, no TLS to manage. Tailscale's identity (you, signed in via Google/GitHub/SSO) is the access control.

## What you need

- A Linux VM with Docker or Podman installed. Hetzner CX22 (€4/mo, 4 GB RAM) is enough; a Raspberry Pi 5 at home works the same way.
- A free [Tailscale account](https://tailscale.com).
- A [Tailscale auth key](https://login.tailscale.com/admin/settings/keys) — generate one with the "ephemeral" toggle off if you want the node to persist.
- About 5 minutes.

## One-line install

SSH into the VM, then run:

```sh
curl -fsSL https://raw.githubusercontent.com/simonnordberg/veckomenyn/main/install.sh \
  | TS_AUTHKEY=tskey-... sh
```

What it does:

1. Detects `podman` (or falls back to `docker`).
2. Downloads `docker-compose.yml` and `docker-compose.tailscale.yml` from the repo into `~/veckomenyn`.
3. Writes a `.env` with your auth key (mode `600`, never read by anything but Compose).
4. Runs `compose -f docker-compose.yml -f docker-compose.tailscale.yml up -d`.

The Tailscale sidecar joins your tailnet under the hostname `veckomenyn`. The app container shares its network namespace, so the only way to reach it is through the tailnet — direct connections to the VM's public IP get nothing.

Open the printed URL from any of your devices that are signed into the tailnet:

```
http://veckomenyn.<your-tailnet>.ts.net:8080
```

(Find your tailnet name in the [Tailscale admin panel](https://login.tailscale.com/admin).)

The first-run setup wizard takes you the rest of the way.

## Manual install

Skip the script if you'd rather see exactly what lands. The same effect:

```sh
mkdir -p ~/veckomenyn && cd ~/veckomenyn

curl -O https://raw.githubusercontent.com/simonnordberg/veckomenyn/main/docker-compose.yml
curl -O https://raw.githubusercontent.com/simonnordberg/veckomenyn/main/docker-compose.tailscale.yml

echo "TS_AUTHKEY=tskey-..." > .env
chmod 600 .env

podman compose -f docker-compose.yml -f docker-compose.tailscale.yml up -d
```

## What just happened

`docker-compose.tailscale.yml` overlays two changes on the base compose:

1. Adds a `tailscale` service running the official `tailscale/tailscale` image. It joins the tailnet using `TS_AUTHKEY`, persists state in a named volume so reboots don't re-auth.
2. Sets `network_mode: service:tailscale` on the `app` service, putting it in the sidecar's network namespace. Removes the `ports:` mapping so the host has nothing exposed.

The base `docker-compose.yml` is unchanged. Run it without the overlay and you have the LAN deployment described in [Quickstart](quickstart.md). Layer the overlay back on whenever you want the tailnet path again.

## Updating

```sh
cd ~/veckomenyn
podman compose -f docker-compose.yml -f docker-compose.tailscale.yml pull
podman compose -f docker-compose.yml -f docker-compose.tailscale.yml up -d
```

The `:0.2` tag in the compose file pins to the patch channel — see [Upgrading](upgrading.md) for choosing a different cadence and for the in-app update banner. Pre-migration `pg_dump` snapshots run on every restart that has pending migrations, so an upgrade can't eat your data.

## Backups on a VM

[Pre-migration snapshots](backups.md) live in `~/veckomenyn/backups/` on the VM and survive `compose down -v`. They don't survive **the VM itself going away** — provider incidents, accidental termination, lost SSH keys to the only admin account. For real safety, sync the backups directory off the VM:

```sh
# nightly cron, push to Backblaze B2 / S3 / Hetzner Storage Box / your home
0 4 * * * restic -r b2:bucket:/veckomenyn backup ~/veckomenyn/backups
```

Use whatever you already trust — `restic`, `rclone`, `borg`, `rsync` to a second host. The directory is plain dump files, no special handling needed.

## Tradeoffs

- **The VM is the only host.** If it dies, you restore from off-host backups onto a new one. If you want zero downtime, that's a different deployment shape and probably the wrong tool — Veckomenyn is a household app, not a service.
- **Tailscale free tier is enough.** 100 devices is more than a household will ever use. Personal/business plans add features you won't need here.
- **You trust Tailscale.** They handle auth and key exchange. If that bothers you, run [Headscale](https://github.com/juanfont/headscale) for the same compose pattern with a self-hosted control plane.
- **Family members need Tailscale on their devices.** Free, ~5 min per device. Worth it for the security model.

For the threat model details and what the app encrypts on disk, see [SECURITY.md](../SECURITY.md).
