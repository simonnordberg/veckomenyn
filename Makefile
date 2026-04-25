.PHONY: build build-server build-import build-import-week build-web dev test lint fmt verify install-hooks clean

# Stamped into the main binary via -ldflags. CI overrides VERSION/COMMIT/BUILT_AT
# from semver tags; local builds derive from git so the binary still reports
# something useful at /api/version.
VERSION  ?= $(shell git describe --tags --dirty --always 2>/dev/null || echo dev)
COMMIT   ?= $(shell git rev-parse HEAD 2>/dev/null || echo unknown)
BUILT_AT ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS  := -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.builtAt=$(BUILT_AT)

build: build-web build-server build-import build-import-week

build-server: build-web
	go build -ldflags "$(LDFLAGS)" -o bin/veckomenyn ./cmd/veckomenyn

build-import:
	go build -o bin/veckomenyn-import ./cmd/veckomenyn-import

build-import-week:
	go build -o bin/veckomenyn-import-week ./cmd/veckomenyn-import-week

build-web:
	cd web && pnpm install --frozen-lockfile && pnpm build

# Run the backend and frontend together with hot-reload.
# Requires Postgres to be reachable via DATABASE_URL (e.g. `podman compose up -d db`; `docker compose` also works).
dev:
	@(go run ./cmd/veckomenyn & cd web && pnpm dev) 2>&1

test:
	go test -race ./...
	cd web && pnpm typecheck

lint:
	golangci-lint run
	cd web && pnpm lint

fmt:
	go fmt ./...
	cd web && pnpm format

# Mirrors .github/workflows/ci.yml exactly. Run before pushing if you
# want to know what CI is going to say.
verify:
	go build ./...
	go test -race ./...
	golangci-lint run
	cd web && pnpm install --frozen-lockfile
	cd web && pnpm exec biome ci .
	cd web && pnpm typecheck
	cd web && pnpm build

# One-time: point git at the tracked hooks in .githooks/.
install-hooks:
	git config core.hooksPath .githooks
	@echo "Pre-push hook installed. Runs 'make verify' before every push."

clean:
	rm -rf bin web/dist/assets web/dist/index.html web/*.tsbuildinfo
