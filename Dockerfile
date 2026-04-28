# syntax=docker/dockerfile:1.7

# Base images come from Docker Hub. Anonymous limit (100 pulls / 6h / IP)
# is plenty for our release cadence; ECR Public's 1 GB/hour anonymous
# limit was tighter in practice and broke the release on busy GitHub
# Actions runners.

# ---- Web frontend ---------------------------------------------------------
FROM node:24-alpine AS web
WORKDIR /app/web
RUN corepack enable pnpm
COPY web/package.json web/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile
COPY web/ ./
RUN pnpm build

# ---- Go binaries ----------------------------------------------------------
FROM golang:1.26-alpine AS go
WORKDIR /app
RUN apk add --no-cache ca-certificates
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /app/web/dist ./web/dist
# Build metadata stamped into the main binary via -ldflags. CI passes real
# values from the release workflow; local builds fall back to the defaults
# baked into main.go.
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILT_AT=unknown
ENV CGO_ENABLED=0
RUN go build -trimpath \
        -ldflags="-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.builtAt=${BUILT_AT}" \
        -o /out/veckomenyn ./cmd/veckomenyn

# ---- Runtime --------------------------------------------------------------
FROM alpine:3.23
# postgresql17-client gives us pg_dump for the in-process pre-migration
# snapshot. Pin major to match docker-compose.yml's db service — clients
# newer than the server work, but the major must match what the data was
# written by. See CONTRIBUTING.md "release checklist".
RUN apk add --no-cache ca-certificates tzdata postgresql17-client
COPY --from=go /out/veckomenyn /usr/local/bin/veckomenyn
# Runs as root in-container by default. For rootless podman that's the
# host user (no privilege escalation). For docker rootful it lets the
# pre-migration pg_dump write into the bind-mounted ./backups dir
# without UID-mapping gymnastics. Override with --user if you have
# specific isolation needs.
EXPOSE 8080
ENTRYPOINT ["veckomenyn"]
