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

## Commits

Conventional Commits. `feat:`, `fix:`, `refactor:`, `docs:`, `chore:`, `test:`. One logical change per commit. Reference issues where useful.

## Security

Suspected vulnerabilities don't go in public issues. See [SECURITY.md](SECURITY.md).
