<p align="center">
  <img src="docs/logo.svg" alt="Veckomenyn" width="420" />
</p>

<p align="center"><em>Familjens veckomeny, planerad och handlad.</em></p>

---

A Claude agent plans your family's week of dinners and builds the grocery cart. Self-hosted. It learns what's in the fridge, which kid won't eat cilantro, which store brands you trust.

Shopping backends are pluggable. Willys.se ships today.

## The loop

1. Set household constraints. Dinners per week, servings, allergies, what's usually in the pantry.
2. Ask the agent to plan a week. Swap dishes, regenerate, nudge until it looks right.
3. Let the agent build the grocery cart. It aggregates ingredients across all dinners, picks one product per ingredient, verifies.
4. Place the order in the store's own UI. Veckomenyn stops at cart-ready. Delivery and payment stay where they belong.
5. After the week, record a retrospective. That feedback shapes next week.

![Weekly menu](docs/screenshots/week-light.png)

## Run it

```sh
cp .env.example .env
echo "MASTER_KEY=$(openssl rand -base64 32)" >> .env
podman compose up -d
open http://localhost:8080
```

That pulls `ghcr.io/simonnordberg/veckomenyn:0.1` — the patch channel for the 0.1 line. Podman is the default engine; `docker compose up -d` works the same way. The compose file is plain OCI.

### Upgrading

The image tag in `docker-compose.yml` is your update channel. Pick the discipline you want:

| Tag | What you get | When to use |
|---|---|---|
| `:0.1.3` | Pinned exact version. Never moves. | Maximum stability. You bump manually. |
| `:0.1` | Latest 0.1.x patch. | **Default.** Bug fixes only, no surprise feature changes. |
| `:0` | Latest 0.x. | Patches + new features within 0.x. Breaking changes (1.0) require a deliberate bump. |
| `:latest` | Whatever just shipped. | You like surprises. |

To upgrade on your channel:

```sh
podman compose pull && podman compose up -d
```

Pre-migration snapshots are automatic — see [Backups](#backups). Watch [GitHub Releases](https://github.com/simonnordberg/veckomenyn/releases) for breaking-change notes.

> **Do not expose port 8080 to the public internet.** There is no authentication. Anyone who can reach the port can read your preferences, order history, and stored credentials, and can spend your Anthropic balance. Run it on a trusted LAN or behind Tailscale / VPN. See [Threat model](#threat-model).

Open Settings. Add an Anthropic API key and store credentials. Both encrypt at rest with AES-256-GCM using `MASTER_KEY`, and the API masks them with a per-process sentinel on read.

Seed starter preferences (optional):

```sh
podman compose exec app veckomenyn-import --from /usr/local/share/veckomenyn/preferences
```

Any directory of `.md` files works. One file per category.

## Screenshots

| Week view (light) | Week view (dark) |
|---|---|
| ![](docs/screenshots/week-light.png) | ![](docs/screenshots/week-dark.png) |
| **Chat drawer.** The agent narrates what it's doing. | **Chat** (dark). |
| ![](docs/screenshots/chat-open.png) | ![](docs/screenshots/chat-dark.png) |
| **Settings.** Household defaults and integrations. | **Settings** (dark). |
| ![](docs/screenshots/settings.png) | ![](docs/screenshots/settings-dark.png) |
| **Preferences.** Free-form markdown per category. | **Print preview.** Paper stays light whatever the theme. |
| ![](docs/screenshots/preferences.png) | ![](docs/screenshots/print-preview.png) |

## Layout

```
cmd/
  veckomenyn/              HTTP server + embedded SPA. Main binary.
  veckomenyn-import/       Seeds preference files.
  veckomenyn-import-week/  Imports a historical week from markdown + CSV.
internal/
  agent/            Claude agent. System prompt, tools, streaming loop.
  willys/           Willys.se HTTP client.
  shopping/         Store-agnostic Provider interface. Willys adapter.
  providers/        Registry for LLM and shopping backends. AES-GCM at rest.
  server/           chi router, SSE chat, handlers.
  store/            pgxpool + goose migration runner.
  migrations/       Embedded SQL migrations.
web/                React 19, TypeScript, Tailwind v4, Biome, Vite.
shared-data/        Template preferences shipped for new installs.
```

The Go binary embeds the Vite bundle via `//go:embed`.

## Development

```sh
podman compose up -d db   # Postgres only (docker compose also works)
make dev                  # server + frontend with HMR
make test                 # go test -race + frontend typecheck
make lint                 # golangci-lint + biome
```

## Configuration

| Var | Purpose |
|---|---|
| `MASTER_KEY` | 32-byte base64 AES key. Encrypts provider secrets (API keys, store credentials, session cookies) in the DB. Generate with `openssl rand -base64 32`. Required for any real use; if unset the server boots in cleartext mode with a warning. |
| `DATABASE_URL` | Postgres DSN. Set automatically by compose. |
| `HTTP_ADDR` | Listen address. Defaults to `:8080`. |
| `HOST_PORT` | Host port mapped to the container's 8080. Defaults to 8080. |

Everything else, including the Anthropic model, lives in Settings.

## Backups

**Pre-migration snapshots are automatic.** Before applying any pending migration, the app runs `pg_dump --format=custom` into `./backups/` on the host. A bad migration can't eat your data — there's always a snapshot from the previous version sitting next to it. The last 10 are retained (override with `PREMIGRATION_BACKUP_KEEP`). Files are named `{timestamp}_pre-migration_{version}.dump` and are bind-mounted from the host, so `docker compose down -v` (which wipes the DB) leaves them untouched.

Restore from any snapshot:

```sh
podman compose exec -T db pg_restore \
  --clean --if-exists --no-owner --no-privileges \
  -U veckomenyn -d veckomenyn \
  < backups/20260425T100000Z_pre-migration_0.1.0.dump
```

**Scheduled nightly backups (optional)** add a sidecar with daily/weekly/monthly rotation (14/8/6):

```sh
podman compose --profile backup up -d
```

That sidecar is [prodrigestivill/postgres-backup-local](https://github.com/prodrigestivill/docker-postgres-backup-local) and writes to the same `./backups/` directory in its own subfolders. Override `SCHEDULE` or `BACKUP_KEEP_*` in `docker-compose.yml` to change retention.

## Threat model

Single-household LAN deployment. No user accounts, no auth. The network boundary (Tailscale, home VPN, firewall) is what restricts access. Exposing it to the public internet without auth in front is outside scope. See [SECURITY.md](SECURITY.md) for the full picture.

## Contributing

[CONTRIBUTING.md](CONTRIBUTING.md) for PRs. [SECURITY.md](SECURITY.md) for vulnerabilities.

## License

MIT.
