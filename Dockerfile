# syntax=docker/dockerfile:1.6

# =================================================================
# Final Multi-Stage Dockerfile
#
# This file prioritizes:
# 1. Reproducibility: Pinned base images and git-based metadata.
# 2. Minimal Production Size: Using a distroless base for the final image.
# 3. Security: Dedicated non-root users and explicit port mapping.
# =================================================================

# =================================================================
# Global Build Arguments
# =================================================================
ARG GO_VERSION=1.24.5
ARG TARGETPLATFORM=linux/amd64
# Arguments for reproducible metadata passed in during build
ARG VERSION
ARG GIT_COMMIT
ARG SOURCE_URL="https://github.com/vlad/burstgridgo"

# =================================================================
# Builder Stage
# =================================================================
FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine3.20@sha256:d9bde7c9e377f3e498c36004523315573489ce4c949c59508d55c703c0049090 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY ./cmd/ ./cmd/
COPY ./internal/ ./internal/
COPY ./modules/ ./modules/

# Build with validated, reproducible metadata.
RUN BUILD_DATE=$(date -u +'%Y-%m-%dT%H:%M:%SZ') && \
    CGO_ENABLED=0 go build \
    -ldflags=" \
      -s -w \
      -X 'main.version=${VERSION:-v0.0.0-unknown}' \
      -X 'main.commit=${GIT_COMMIT:-none}' \
      -X 'main.buildDate=${BUILD_DATE}' \
    " \
    -o /out/burstgridgo ./cmd/cli

# =================================================================
# Production Stage
# =================================================================
# Re-declare ARGs to make them available in this stage for labels.
ARG VERSION
ARG GIT_COMMIT
ARG SOURCE_URL
ARG BUILD_DATE=$(date -u +'%Y-%m-%dT%H:%M:%SZ')

# Use a minimal distroless image for a tiny, secure production artifact.
FROM gcr.io/distroless/static-debian12:nonroot@sha256:ac8e9320292e4995f5ad4c0529cd6d164a66a350c377953259a584045f0962b1 AS prod

LABEL org.opencontainers.image.created=$BUILD_DATE \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.revision="${GIT_COMMIT}" \
      org.opencontainers.image.source="${SOURCE_URL}"

# Expose a default port for discoverability.
ENV APP_PORT=8080
EXPOSE 8080

COPY --from=builder /out/burstgridgo /burstgridgo

# The distroless 'nonroot' user is sufficient and secure.
USER nonroot

# With distroless, the healthcheck must be self-contained in the binary.
HEALTHCHECK --interval=15s --timeout=3s --start-period=5s --retries=3 \
  CMD ["/burstgridgo", "--healthcheck"]

ENTRYPOINT ["/burstgridgo"]

# =================================================================
# Debug Stage
# =================================================================
# This stage uses a full OS for robust debugging tools,
# acknowledging the trade-off that its environment differs from prod.
FROM debian:12-slim@sha256:637774a44f603c14a24f114b78e4a77953259837775551392686121f6a1936c9 AS debug

ARG UID=10001
# Robustly create a non-root user and group.
RUN groupadd --system --gid ${UID} appgroup && \
    useradd --system --uid ${UID} --gid appgroup appuser

# Install debugging tools.
RUN apt-get update && apt-get install -y --no-install-recommends \
      curl \
      procps \
      strace \
      lsof \
      && rm -rf /var/lib/apt/lists/*

ENV APP_PORT=8080
EXPOSE 8080

COPY --from=builder --chown=appuser:appgroup /out/burstgridgo /usr/local/bin/burstgridgo

USER appuser

# A more robust healthcheck is possible here because we have curl.
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

# Pre-fetch dependencies and install the hot-reload tool.
COPY --chown=devuser:devuser go.mod go.sum ./
RUN go mod download && \
    go install github.com/cosmtrek/air@latest

# To run: docker run --rm -it -u "$(id -u):$(id -g)" -v .:/app <image>
CMD ["air"]