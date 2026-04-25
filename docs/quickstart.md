# Quickstart (LAN)

One command on a host with Docker or Podman. **For access from outside your home network, use the [Tailscale guide](deploy-tailscale.md) instead.** Veckomenyn has no built-in authentication and must not be exposed to the public internet.

## Run it

```sh
podman compose up -d
open http://localhost:8080
```

That pulls `ghcr.io/simonnordberg/veckomenyn:0.3` from GHCR, the patch channel for the 0.3 line. The app generates and persists its own AES master key on first boot, so there's no `.env` ritual. Podman is the default; `docker compose up -d` works the same way.

The first request triggers a setup wizard:

1. LAN-only warning.
2. Anthropic API key ([console.anthropic.com](https://console.anthropic.com/settings/keys)).
3. Optional starter preferences. Anonymised templates for cooking style, family routines, sourcing rules. Editable later.

## What's running

- **`db`**: Postgres 17 with meal plans, preferences, conversations, ratings.
- **`app`**: Go binary serving the API and embedded React SPA.

Persistent state:

- `pgdata` (Docker named volume): the database.
- `./backups` (bind mount): automatic pre-migration `pg_dump` snapshots, plus any manual or nightly backups. Survives `docker compose down -v`.

## Day 2

- **[Upgrading](upgrading.md)**. Channels, the in-app banner, automatic snapshots.
- **[Backups & restore](backups.md)**. What's automatic, how to restore.
- **[Configuration](configuration.md)**. Environment variables.

## Threat model

No authentication. Anyone reaching port 8080 reads your data and spends your Anthropic balance. Run on a trusted LAN, or follow the [Tailscale guide](deploy-tailscale.md). Full picture in [SECURITY.md](../SECURITY.md).
