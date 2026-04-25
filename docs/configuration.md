# Configuration

Env vars only cover what the binary needs before the UI loads. Anthropic model, store credentials, household defaults, and backup schedule all live in Settings.

| Var | Purpose |
|---|---|
| `MASTER_KEY` | 32-byte base64 AES key encrypting provider secrets in the DB. **Optional**: auto-generated and persisted on first boot. Set explicitly only to manage the key externally (KMS, sealed secrets). Generate with `openssl rand -base64 32`. |
| `DATABASE_URL` | Postgres DSN. Set automatically by compose. |
| `HTTP_ADDR` | Listen address. Defaults to `:8080`. |
| `HOST_PORT` | Host port mapped to the container's 8080. Defaults to 8080. |
| `BACKUP_DIR` | Where pre-migration `pg_dump` snapshots are written. Set by compose to `/var/lib/veckomenyn/backups`. Empty disables snapshots. |
| `PREMIGRATION_BACKUP_KEEP` | Number of pre-migration snapshots to retain. Defaults to 10. |
| `VECKOMENYN_SKIP_PREMIGRATION_BACKUP` | Set to `1` to allow migration even if the pre-migration `pg_dump` fails. Dev-only escape hatch. |
| `DISABLE_UPDATE_CHECK` | Set to `1` to opt out of polling GitHub releases for the in-app update banner. |
| `TS_AUTHKEY` | Only used by the Tailscale overlay (`docker-compose.tailscale.yml`). Tailscale auth key for joining the tailnet. |
