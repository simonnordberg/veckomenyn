# Quickstart (LAN)

The fastest path: a single command on your home network. Works on any host with Docker or Podman — a laptop, a mini-PC, a Raspberry Pi 5, a homelab VM. **For remote access from outside your home, use the [Tailscale guide](deploy-tailscale.md) instead** — Veckomenyn has no built-in authentication and must not be exposed to the public internet.

## Run it

```sh
podman compose up -d
open http://localhost:8080
```

That pulls `ghcr.io/simonnordberg/veckomenyn:0.3` from GHCR — the patch channel for the 0.3 line. The app generates and persists its own AES master key on first boot, so there's no `.env` ritual to start. Podman is the default; `docker compose up -d` works the same way. The compose file is plain OCI.

The first time you open the URL, a setup wizard walks you through:

1. A reminder that the app has no authentication and must stay on a trusted network.
2. Pasting your [Anthropic API key](https://console.anthropic.com/settings/keys).
3. Optionally seeding starter preferences (cooking style, family routines, sourcing rules) — anonymised templates you can edit later.

Then you're in.

## What's running

Two containers:

- **`db`** — Postgres 17 holding meal plans, preferences, conversations, ratings.
- **`app`** — the Go binary serving the API and the embedded React SPA.

Persistent state lives in two places:

- `pgdata` (Docker named volume) — the database itself.
- `./backups` (bind-mounted from the project directory) — automatic pre-migration `pg_dump` snapshots and any manual/nightly backups you take. Survives `docker compose down -v`.

## Day-2

- **[Upgrading](upgrading.md)** — pick an update channel, watch for new releases, the in-app banner.
- **[Backups & restore](backups.md)** — what's automatic, what's optional, how to restore.
- **[Configuration](configuration.md)** — environment variables for tweaking behaviour.

## Don't put it on the public internet

There is no authentication. Anyone who can reach port 8080 can read your data, your preferences, and your stored credentials, and can spend your Anthropic balance. Run on a trusted LAN, or follow [the Tailscale guide](deploy-tailscale.md) for safe remote access. Threat model details are in [SECURITY.md](../SECURITY.md).
