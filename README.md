<p align="center">
  <img src="docs/logo.svg" alt="Veckomenyn" width="420" />
</p>

<p align="center"><em>Familjens veckomeny, planerad och handlad.</em></p>

---

A self-hosted web app that plans a family's week of dinners, builds the grocery cart, and remembers what worked. The agent learns preferences over time. What's already in the fridge, which kid won't eat cilantro, which store brands the household trusts.

Shopping backends are pluggable. Willys.se is the one that ships today.

## The loop

1. Set household constraints and preferences. Dinners per week, servings, allergies, what's usually in the pantry.
2. Ask the agent to plan a week. Swap dishes, regenerate, nudge until it looks right.
3. Let the agent build the grocery cart. It aggregates ingredients across all dinners, picks one product per ingredient, verifies.
4. Place the order in the store's own UI. Veckomenyn stops at cart-ready. Delivery and payment stay where they belong.
5. After the week, record a retrospective. That feedback shapes next week.

![Weekly menu](docs/screenshots/week-light.png)

## Run it

```sh
cp .env.example .env
echo "MASTER_KEY=$(openssl rand -base64 32)" >> .env
docker compose up -d
open http://localhost:8080
```

`podman-compose up -d` works too. The compose file is plain OCI.

Open Settings, add an Anthropic API key and store credentials. Both get AES-256-GCM encrypted at rest when `MASTER_KEY` is set. The API responses mask them with a random per-process sentinel, so no secrets touch the wire.

To seed the included starter preferences:

```sh
docker compose exec app veckomenyn-import --from /usr/local/share/veckomenyn/preferences
```

Any directory of `.md` files works. One file, one preference category.

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
  veckomenyn/              HTTP server + embedded SPA. The main binary.
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
web/                React 19 + TypeScript + Tailwind v4 + Biome (Vite).
shared-data/        Template preferences shipped for new installs.
```

The Go binary embeds the Vite bundle via `//go:embed`. One process, one image.

## Development

```sh
docker compose up -d db   # Postgres only
make dev                  # server + frontend with HMR

make test                 # go test -race + frontend typecheck
make lint                 # golangci-lint + biome
```

## Configuration

Container env:

| Var | Purpose |
|---|---|
| `MASTER_KEY` | 32-byte base64 AES key. Encrypts provider secrets in the DB. Generate with `openssl rand -base64 32`. Without it, secrets live in cleartext and the server logs a warning at boot. |
| `DATABASE_URL` | Postgres DSN. Set automatically by compose. |
| `HTTP_ADDR` | Listen address. Defaults to `:8080`. |
| `HOST_PORT` | Host port mapped to the container's 8080. Defaults to 8080. |

Everything else, including the Anthropic model choice, lives in Settings.

## Backups

The compose stack runs a sidecar that dumps Postgres nightly to `./backups/` on the host. Retention defaults to 14 daily, 8 weekly, 6 monthly snapshots. Dumps use `--clean --if-exists --no-owner --no-privileges`, so restoring into a fresh database is a one-liner:

```sh
gzip -dc backups/daily/veckomenyn-YYYY-MM-DD.sql.gz \
  | docker compose exec -T db psql -U veckomenyn -d veckomenyn
```

Override `SCHEDULE` or the `BACKUP_KEEP_*` vars in `docker-compose.yml` for different retention. The sidecar is [prodrigestivill/postgres-backup-local](https://github.com/prodrigestivill/docker-postgres-backup-local).

## Threat model

Single-household LAN deployment. No user accounts, no built-in auth. The network boundary (Tailscale, home VPN, firewall) is what restricts access. Exposed to the public internet without auth in front of it is outside scope.

## Contributing and security

- [CONTRIBUTING.md](CONTRIBUTING.md).
- [SECURITY.md](SECURITY.md) for vulnerability reports.

## License

MIT.
