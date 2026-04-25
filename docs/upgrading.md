# Upgrading

Updates are driven from inside the app. Two paths, both backed by the same trigger:

1. **Manual**: when a new version is published, the update banner shows it. Click **Update now**. The app pulls the new image, restarts itself, and reloads the page once the new version reports back.
2. **Automatic**: Settings → Updates → toggle on **Apply updates automatically**. The app checks once an hour; if a newer version exists, it fires the same trigger.

Both paths require the local Watchtower sidecar to be running (it ships in the default compose). Watchtower is **passive**: no polling, no surprise restarts. It only acts when the app asks it to.

## Channels

The image tag in your `docker-compose.yml` is the channel:

| Tag | Resolves to | When |
|---|---|---|
| `:0.4.0` | Pinned exact version. | Maximum stability. Bump manually. |
| `:0.4` | Latest 0.4.x patch. | **Default.** Bug fixes only. |
| `:0` | Latest 0.x. | Patches + new features within 0.x. |
| `:latest` | Whatever just shipped. | Bleeding edge. |

Edit the `image:` line to switch channels. The next "Update now" click pulls from the new channel.

## What to watch

The app polls GitHub releases hourly. When a newer version exists on the repo, the banner shows "Update available: vX.Y.Z" plus two links:

- **Release notes**: human-curated changelog for the new version.
- **What changed**: GitHub diff between your current version and the latest. Skipped when one of them isn't a stable semver (e.g. `dev` builds).

`DISABLE_UPDATE_CHECK=1` opts out of the polling.

## Manual escape hatch

The Watchtower-backed path is the recommended way. If for some reason you want to skip the app and run the upgrade by hand, the underlying mechanic is still just `compose pull && up -d`:

```sh
podman compose pull && podman compose up -d
```

Same result; pre-migration `pg_dump` runs on the new container's boot regardless.

## Migration safety

Before applying any pending migration, the app takes a `pg_dump` into `./backups/`. If the dump fails, the app refuses to migrate. See [Backups](backups.md) for retention and restore.
