# syntax=docker/dockerfile:1.7
# ============================================================================
# NebulaCloud — backend Dockerfile (multi-stage, multi-target)
#
# Targets:
#   - builder      : compiles every cmd into /out
#   - api          : minimal runtime image for the API
#   - build-worker : minimal runtime image for the build worker
#   - runtime-agent: minimal runtime image for the runtime agent
#   - api-dev      : hot-reload dev image (used by docker-compose.override.yml)
# ============================================================================

ARG GO_VERSION=1.25
# Official Docker CLI image (Alpine). Tags like docker:24-cli-alpine do not exist on Hub.
ARG DOCKER_CLI_IMAGE=docker:27-cli

# ---------------------------------------------------------------------------
# Builder
# ---------------------------------------------------------------------------
FROM golang:${GO_VERSION}-alpine AS builder
WORKDIR /src

RUN apk add --no-cache git ca-certificates tzdata

COPY go.mod ./
COPY go.sum* ./
RUN go mod download || true

COPY . .

ENV CGO_ENABLED=0 GOOS=linux

RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    mkdir -p /out && \
    go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api && \
    go build -trimpath -ldflags="-s -w" -o /out/build-worker ./cmd/build-worker && \
    go build -trimpath -ldflags="-s -w" -o /out/runtime-agent ./cmd/runtime-agent

# ---------------------------------------------------------------------------
# API runtime
# ---------------------------------------------------------------------------
FROM gcr.io/distroless/static-debian12:nonroot AS api
WORKDIR /app
COPY --from=builder /out/api /app/api
COPY --from=builder /src/migrations /app/migrations
USER nonroot:nonroot
EXPOSE 8080 9100
ENTRYPOINT ["/app/api"]

# ---------------------------------------------------------------------------
# Build worker runtime (Docker CLI + git + pack for Cloud Native Buildpacks)
# ---------------------------------------------------------------------------
FROM ${DOCKER_CLI_IMAGE} AS build-worker
RUN apk add --no-cache git ca-certificates libc6-compat curl tar \
 && PACK_VERSION=0.35.1 \
 && ARCH="$(uname -m)" \
 && case "$ARCH" in \
      x86_64)  PACK_URL="https://github.com/buildpacks/pack/releases/download/v${PACK_VERSION}/pack-v${PACK_VERSION}-linux.tgz" ;; \
      aarch64) PACK_URL="https://github.com/buildpacks/pack/releases/download/v${PACK_VERSION}/pack-v${PACK_VERSION}-linux-arm64.tgz" ;; \
      *) echo "unsupported arch: $ARCH" >&2; exit 1 ;; \
    esac \
 && curl -fsSL "$PACK_URL" | tar -xz -C /usr/local/bin pack \
 && chmod +x /usr/local/bin/pack
WORKDIR /app
COPY --from=builder /out/build-worker /app/build-worker
ENTRYPOINT ["/app/build-worker"]

# ---------------------------------------------------------------------------
# Runtime agent (Docker CLI against mounted socket)
# ---------------------------------------------------------------------------
FROM ${DOCKER_CLI_IMAGE} AS runtime-agent
RUN apk add --no-cache ca-certificates libc6-compat
WORKDIR /app
COPY --from=builder /out/runtime-agent /app/runtime-agent
ENTRYPOINT ["/app/runtime-agent"]

# ---------------------------------------------------------------------------
# Dev image with hot reload (used only by docker-compose.override.yml)
# ---------------------------------------------------------------------------
FROM golang:${GO_VERSION}-alpine AS api-dev
WORKDIR /workspace
RUN apk add --no-cache git make curl ca-certificates && \
    go install github.com/air-verse/air@v1.61.5 && \
    go install github.com/go-delve/delve/cmd/dlv@latest
EXPOSE 8080 9100
ENV CGO_ENABLED=0
COPY .air.toml /workspace/.air.toml
CMD ["air", "-c", ".air.toml"]
