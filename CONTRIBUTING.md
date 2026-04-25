# Contributing

This is a small opinionated project. A few notes before a PR.

## Scope

Veckomenyn is a household meal planner. Willys.se is the first shopping backend. Other stores are welcome, but through the provider abstractions (`internal/shopping/Provider`, `internal/providers`). No store-specific code in handlers.

## Local loop

```sh
podman compose up -d db   # docker compose works the same
make dev
```

`go.mod` pins the Go toolchain. `web/` uses pnpm.

To exercise the full container image (the one users actually run), layer the dev override on top of the production compose so it builds from source instead of pulling:

```sh
podman compose -f docker-compose.yml -f docker-compose.dev.yml up --build
```

## Before the PR

```sh
make verify   # runs the full CI pipeline locally
```

`verify` mirrors `.github/workflows/ci.yml` exactly: go build + race tests + golangci-lint + biome ci + typecheck + frontend build. Green here means green in CI.

If you want the same checks to run automatically before every push:

```sh
make install-hooks
```

That points git at `.githooks/`, where a `pre-push` hook runs `make verify` and blocks the push on failure. Skip in a pinch with `git push --no-verify`.

## Style

Go follows `gofmt` and `go vet`. `golangci-lint` is the gate.

TypeScript and React go through Biome. One tool for formatting and linting. No ESLint, no Prettier.

Database changes ship as new goose migrations in `internal/migrations/`. Never edit a committed migration.

Dependencies cost. If a helper does the job, write the helper.

## Release checklist

Users on `:0.1` (patch channel) and `:0` (minor channel) get whatever you tag. Hold the line on what each means or the auto-update story breaks.

- **Patch** (0.1.x): bug fixes, doc updates, perf wins. Migrations must be additive — new tables, new nullable columns, new indexes. No renames, no drops, no required-NOT-NULL on existing columns without a default.
- **Minor** (0.x.0): new features, additive schema changes. Same migration rules as patch.
- **Major** (X.0.0): anything goes. Document the migration path in release notes. If the upgrade requires a manual step, gate the boot path so the binary refuses to start without the user's explicit acknowledgement (env var) — never silently apply destructive changes.

Audit older snapshots before any column-shape change. Pre-migration snapshots are restored against a freshly-migrated schema; if your migration drops or renames a column an older snapshot referenced, restore from that snapshot will fail. When in doubt, stage the change across two minor releases (add new column, dual-write, deprecate old).

The `pg_dump` major version bundled in the image must match the `db` service's Postgres major. Bump them together in the same release if you ever change Postgres versions.

## Commits

Conventional Commits. `feat:`, `fix:`, `refactor:`, `docs:`, `chore:`, `test:`. One logical change per commit. Reference issues where useful.

## Security

Suspected vulnerabilities don't go in public issues. See [SECURITY.md](SECURITY.md).
