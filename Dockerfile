# syntax=docker/dockerfile:1.6

# =================================================================
# Global Build Arguments
# =================================================================
ARG GO_VERSION=1.24.5
ARG TARGETPLATFORM=linux/amd64
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
ARG VERSION
ARG GIT_COMMIT
ARG SOURCE_URL
ARG BUILD_DATE=$(date -u +'%Y-%m-%dT%H:%M:%SZ')

# Fully pinned distroless image for maximum reproducibility and minimal size.
FROM gcr.io/distroless/static-debian12:nonroot@sha256:ac8e9320292e4995f5ad4c0529cd6d164a66a350c377953259a584045f0962b1 AS prod

LABEL org.opencontainers.image.created=$BUILD_DATE \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.revision="${GIT_COMMIT}" \
      org.opencontainers.image.source="${SOURCE_URL}"

ENV APP_PORT=8080
EXPOSE 8080

COPY --from=builder /out/burstgridgo /burstgridgo
USER nonroot

HEALTHCHECK --interval=15s --timeout=3s --start-period=5s --retries=3 \
  CMD ["/burstgridgo", "--healthcheck"]

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

# This stage pre-installs dependencies and tools. The source code itself
# should be mounted as a volume for live-reloading, so it is not copied here.
COPY --chown=devuser:devuser go.mod go.sum ./
RUN go mod download && \
    go install github.com/cosmtrek/air@latest

ENTRYPOINT ["air"]