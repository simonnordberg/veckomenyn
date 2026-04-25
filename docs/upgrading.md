# Upgrading

The image tag in your `docker-compose.yml` is the update channel:

| Tag | Resolves to | When |
|---|---|---|
| `:0.3.0` | Pinned exact version. | Maximum stability. Bump manually. |
| `:0.3` | Latest 0.3.x patch. | **Default.** Bug fixes only. |
| `:0` | Latest 0.x. | Patches + new features within 0.x. |
| `:latest` | Whatever just shipped. | Bleeding edge. |

To upgrade:

```sh
podman compose pull && podman compose up -d
```

## Automatic updates

`--profile managed` adds a [Watchtower](https://github.com/containrrr/watchtower) sidecar. It polls GHCR daily and restarts the app on each new image:

```sh
podman compose --profile managed up -d
```

[Pre-migration snapshots](backups.md) run regardless of how the upgrade is triggered.

## In-app banner

The app polls GitHub releases hourly. When a newer version exists, a banner above the topbar links to the release notes and copies the upgrade command. `DISABLE_UPDATE_CHECK=1` opts out.

## Migration safety

Before applying any pending migration the app takes a `pg_dump` into `./backups/`. If the dump fails, the app refuses to migrate. See [Backups](backups.md) for retention and restore.
