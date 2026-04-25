# syntax=docker/dockerfile:1.7

# Base images come from ECR Public (mirror of Docker Official Images) to
# avoid Docker Hub's anonymous pull rate limit. `docker login docker.io`
# works as a fallback if you ever need a tag that isn't mirrored here.

# ---- Web frontend ---------------------------------------------------------
FROM public.ecr.aws/docker/library/node:24-alpine AS web
WORKDIR /app/web
RUN corepack enable pnpm
COPY web/package.json web/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile
COPY web/ ./
RUN pnpm build

# ---- Go binaries ----------------------------------------------------------
FROM public.ecr.aws/docker/library/golang:1.26-alpine AS go
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
        -o /out/veckomenyn ./cmd/veckomenyn && \
    go build -trimpath -ldflags="-s -w" -o /out/veckomenyn-import ./cmd/veckomenyn-import && \
    go build -trimpath -ldflags="-s -w" -o /out/veckomenyn-import-week ./cmd/veckomenyn-import-week

# ---- Runtime --------------------------------------------------------------
FROM public.ecr.aws/docker/library/alpine:3.23
RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S veckomenyn && adduser -S veckomenyn -G veckomenyn
COPY --from=go /out/veckomenyn /usr/local/bin/veckomenyn
COPY --from=go /out/veckomenyn-import /usr/local/bin/veckomenyn-import
COPY --from=go /out/veckomenyn-import-week /usr/local/bin/veckomenyn-import-week
COPY --from=go /app/shared-data /usr/local/share/veckomenyn
USER veckomenyn
EXPOSE 8080
ENTRYPOINT ["veckomenyn"]
