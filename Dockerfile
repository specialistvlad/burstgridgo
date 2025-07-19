# syntax=docker/dockerfile:1.6

# =================================================================
# Definitive Production-Grade Multi-Stage Dockerfile
# =================================================================

# =================================================================
# Global Build Arguments
# =================================================================
ARG GO_VERSION=1.24.5
ARG TARGETPLATFORM=linux/amd64
# Metadata ARGs are passed in from the CI/CD pipeline for a single source of truth
ARG VERSION
ARG GIT_COMMIT
ARG BUILD_DATE
ARG SOURCE_URL="https://github.com/vlad/burstgridgo"
ARG LICENSE="MIT"

# =================================================================
# Builder Stage
# =================================================================
FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine3.20@sha256:d9bde7c9e377f3e498c36004523315573489ce4c949c59508d55c703c0049090 AS builder

# TARGETOS and TARGETARCH are automatically supplied by buildx via TARGETPLATFORM
ARG TARGETOS
ARG TARGETARCH
ARG VERSION
ARG GIT_COMMIT
ARG BUILD_DATE

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY ./cmd/ ./cmd/
COPY ./internal/ ./internal/
COPY ./modules/ ./modules/

# Build a true cross-platform binary using GOOS and GOARCH
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -ldflags=" \
    -s -w \
    -X 'main.version=${VERSION:-v0.0.0-unknown}' \
    -X 'main.commit=${GIT_COMMIT:-none}' \
    -X 'main.buildDate=${BUILD_DATE:-unavailable}' \
    " \
    -o /out/burstgridgo ./cmd/cli

# =================================================================
# Production Stage
# =================================================================
ARG VERSION
ARG GIT_COMMIT
ARG BUILD_DATE
ARG SOURCE_URL
ARG LICENSE

# Use the 'cc' variant of distroless to include curl for a robust healthcheck.
FROM gcr.io/distroless/cc-debian12@sha256:a8ac44336034237c95a09c3132e4242d59d182a4505c5579f3cec3a63447332c AS prod

LABEL org.opencontainers.image.created=$BUILD_DATE \
    org.opencontainers.image.version="${VERSION}" \
    org.opencontainers.image.revision="${GIT_COMMIT}" \
    org.opencontainers.image.source="${SOURCE_URL}" \
    org.opencontainers.image.licenses="${LICENSE}"

ENV APP_PORT=8080
EXPOSE 8080

COPY --from=builder /out/burstgridgo /burstgridgo
USER nonroot

# Uniform, robust healthcheck.
HEALTHCHECK --interval=15s --timeout=3s --start-period=5s --retries=3 \
    CMD ["curl", "--fail", "http://localhost:8080/health"]

ENTRYPOINT ["/burstgridgo"]

# =================================================================
# Debug Stage
# =================================================================
FROM debian:12-slim@sha256:637774a44f603c14a24f114b78e4a77953259837775551392686121f6a1936c9 AS debug

ARG UID=10001
RUN groupadd --system --gid ${UID} appgroup && \
    useradd --system --uid ${UID} --gid appgroup appuser

RUN apt-get update && apt-get install -y --no-install-recommends \
    curl procps strace lsof \
    && rm -rf /var/lib/apt/lists/*

ENV APP_PORT=8080
EXPOSE 8080

COPY --from=builder --chown=appuser:appgroup /out/burstgridgo /usr/local/bin/burstgridgo
USER appuser

# Healthcheck is now identical to production.
HEALTHCHECK --interval=15s --timeout=3s --start-period=5s --retries=3 \
    CMD curl --fail http://localhost:${APP_PORT}/health || exit 1

ENTRYPOINT ["/usr/local/bin/burstgridgo"]

# =================================================================
# Development Stage
# =================================================================
FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine3.20@sha256:d9bde7c9e377f3e498c36004523315573489ce4c949c59508d55c703c0049090 AS dev

ARG UID=10002
RUN adduser -D -u ${UID} devuser

USER devuser
WORKDIR /app

ENV APP_PORT=8080
EXPOSE 8080

COPY --chown=devuser:devuser go.mod go.sum ./
RUN go mod download && \
    go install github.com/cosmtrek/air@latest

ENTRYPOINT ["air"]
