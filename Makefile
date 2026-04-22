.PHONY: build build-server build-import build-import-week build-web dev test lint fmt clean

build: build-web build-server build-import build-import-week

build-server: build-web
	go build -o bin/veckomenyn ./cmd/veckomenyn

build-import:
	go build -o bin/veckomenyn-import ./cmd/veckomenyn-import

build-import-week:
	go build -o bin/veckomenyn-import-week ./cmd/veckomenyn-import-week

build-web:
	cd web && pnpm install --frozen-lockfile && pnpm build

# Run the backend and frontend together with hot-reload.
# Requires Postgres to be reachable via DATABASE_URL (e.g. `docker compose up -d db`).
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

clean:
	rm -rf bin web/dist/assets web/dist/index.html web/*.tsbuildinfo
