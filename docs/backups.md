# Backups & restore

Two layers, both automatic.

## Pre-migration snapshots

Before applying any pending migration, the app runs `pg_dump --format=custom` into `./backups/` on the host. A bad migration can't eat your data; there's always a snapshot from the previous version next to it.

- Filenames: `{timestamp}_pre-migration_{version}.dump`.
- Retention: last 10 (override with `PREMIGRATION_BACKUP_KEEP`).
- Bind-mounted, so `docker compose down -v` doesn't touch them.

If `pg_dump` fails, the app refuses to migrate. `VECKOMENYN_SKIP_PREMIGRATION_BACKUP=1` bypasses (dev only).

## In-app backups

Settings → Backups lists every snapshot (pre-migration, manual, nightly) with download and delete per row, plus:

- **Take backup now**: manual snapshot on demand.
- **Nightly automatic backup**: toggle on, retention configurable.

All snapshots land in `./backups/` and download from the UI as single files, so off-host stashing doesn't need shell access.

## Off-host backups

`./backups/` is plain files on the host. Sync nightly with `restic`, `rclone`, `rsync`, or `borg`. Whichever you already use.

## Restore

Any snapshot is a stock `pg_dump --format=custom` archive:

```sh
podman compose exec -T db pg_restore \
  --clean --if-exists --no-owner --no-privileges \
  -U veckomenyn -d veckomenyn \
  < backups/20260425T100000Z_pre-migration_0.2.0.dump
```

After `docker compose down -v`, run `docker compose up -d` first so migrations create the empty schema; then `pg_restore` with `--clean --if-exists` swaps in the snapshot's data.

## Encryption note

Provider credentials and Willys session cookies are AES-GCM-wrapped at rest. The wrapping key lives in `system_secrets` in the same DB and is dumped alongside, so a snapshot is self-contained. Anyone holding the dump file has both ciphertext and key. Treat snapshots as sensitive in transit and at rest.
