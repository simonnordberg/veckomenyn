# Upgrading

The image tag in your `docker-compose.yml` is your update channel. Pick the discipline you want:

| Tag | What you get | When to use |
|---|---|---|
| `:0.2.3` | Pinned exact version. Never moves. | Maximum stability. You bump manually. |
| `:0.2` | Latest 0.2.x patch. | **Default.** Bug fixes only, no surprise feature changes. |
| `:0` | Latest 0.x. | Patches + new features within 0.x. Breaking changes (1.0) require a deliberate bump. |
| `:latest` | Whatever just shipped. | You like surprises. |

To upgrade on your channel:

```sh
podman compose pull && podman compose up -d
```

## Automatic updates

`--profile managed` adds a [Watchtower](https://github.com/containrrr/watchtower) sidecar that polls GHCR daily and restarts the app when a newer image arrives on your channel:

```sh
podman compose --profile managed up -d
```

[Pre-migration snapshots](backups.md) run regardless of how the upgrade is triggered, so an automatic update can't eat your data.

## What to watch

The app polls GitHub releases hourly and shows a banner above the topbar when a newer version is available. Click it to read the release notes and copy the upgrade command. Set `DISABLE_UPDATE_CHECK=1` to opt out of the polling.

For breaking-change notes, watch [GitHub Releases](https://github.com/simonnordberg/veckomenyn/releases) directly.

## Migration safety

Before applying any pending migration, the app takes a `pg_dump` snapshot into `./backups/`. If the dump fails, the app refuses to migrate — see [Backups](backups.md) for retention, restore, and override flags.
