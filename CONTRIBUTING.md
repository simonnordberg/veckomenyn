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

Integration tests for SQL-touching code are gated on `TEST_DATABASE_URL`; they skip silently when unset. To run them locally against the dev compose db:

```sh
TEST_DATABASE_URL='postgres://veckomenyn:veckomenyn@localhost:5432/veckomenyn?sslmode=disable' \
  go test -race ./...
```

CI sets `TEST_DATABASE_URL` against an ephemeral `services.postgres` so PRs always run them.

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

Use one of these to cut a release. The next version is computed from the latest tag, you don't need to track what's current:

```sh
make release        # patch bump (default)
make release-minor  # new feature, additive schema
make release-major  # breaking changes
```

Each command bumps the compose pin and README upgrade-channel examples to match the new minor (no-op for patch bumps), commits, and creates an annotated tag. It does NOT push; the print message tells you the exact two `git push` commands to run.

Users on `:0.x` (patch channel) and `:0` (minor channel) get whatever you tag. Hold the line on what each version step means or the auto-update story breaks.

- **Patch** (0.x.y): bug fixes, doc updates, perf wins. Migrations must be additive (new tables, new nullable columns, new indexes). No renames, no drops, no required-NOT-NULL on existing columns without a default.
- **Minor** (0.x.0): new features, additive schema changes. Same migration rules as patch. The release script bumps `docker-compose.yml` and README example versions automatically; the release workflow refuses to publish if they're out of sync, so a forgotten manual edit can't slip through.
- **Major** (X.0.0): anything goes. Document the migration path in release notes. If the upgrade requires a manual step, gate the boot path so the binary refuses to start without the user's explicit acknowledgement (env var). Never silently apply destructive changes.

Audit older snapshots before any column-shape change. Pre-migration snapshots are restored against a freshly-migrated schema; if your migration drops or renames a column an older snapshot referenced, restore from that snapshot will fail. When in doubt, stage the change across two minor releases (add new column, dual-write, deprecate old).

The `pg_dump` major version bundled in the image must match the `db` service's Postgres major. Bump them together in the same release if you ever change Postgres versions.

## Commits

Conventional Commits. `feat:`, `fix:`, `refactor:`, `docs:`, `chore:`, `test:`. One logical change per commit. Reference issues where useful.

## Security

Suspected vulnerabilities don't go in public issues. See [SECURITY.md](SECURITY.md).
