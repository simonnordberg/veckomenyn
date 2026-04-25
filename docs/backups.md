# Backups & restore

Two layers, both automatic.

## Pre-migration snapshots

Before applying any pending database migration, the app runs `pg_dump --format=custom` into `./backups/` on the host. A bad migration can't eat your data — there's always a snapshot from the previous version sitting next to it.

- Filenames are `{timestamp}_pre-migration_{version}.dump`.
- The last 10 are retained (override with `PREMIGRATION_BACKUP_KEEP`).
- Bind-mounted from the host, so `docker compose down -v` (which wipes the DB) leaves them untouched.

If `pg_dump` fails, the app refuses to migrate. Set `VECKOMENYN_SKIP_PREMIGRATION_BACKUP=1` to bypass — useful in dev, dangerous in production.

## In-app backups

Settings → Backups gives you a listing of every snapshot the system has ever taken (pre-migration, manual, nightly), with download and delete buttons per row, plus:

- **Take backup now** — manual snapshot on demand.
- **Nightly automatic backup** — toggle on to take a daily `pg_dump`. Retention is configurable.

All backups land in the same `./backups/` directory and are downloadable as a single file from the UI, so you can stash them off-host (S3, B2, your laptop) without going into the container.

## Off-host backups

The bind-mounted `./backups/` directory is just regular files on the host. Sync them to wherever you trust with whatever tool you trust — `restic`, `rclone`, `rsync`, `borg`. A nightly cron is enough for a household app.

## Restore

Any snapshot is a stock `pg_dump --format=custom` archive, restorable with stock `pg_restore`:

```sh
podman compose exec -T db pg_restore \
  --clean --if-exists --no-owner --no-privileges \
  -U veckomenyn -d veckomenyn \
  < backups/20260425T100000Z_pre-migration_0.2.0.dump
```

If you wiped the DB volume (`docker compose down -v`), bring it back up first (`docker compose up -d`) so the migrations create the empty schema, then run `pg_restore` with `--clean --if-exists` so the import drops the empty tables and replaces them with the snapshot's data.

## What's encrypted in those dumps

The `providers` and `willys_session` tables hold AES-GCM-wrapped credentials (Anthropic key, Willys password, Willys session cookies). The wrapping key lives in `system_secrets` in the same DB and is dumped alongside, so a snapshot is self-contained — restore it anywhere and the credentials decrypt correctly. Anyone with the dump file has both the ciphertext and the key, so treat snapshots as sensitive material in transit and at rest.
