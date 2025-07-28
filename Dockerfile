# syntax=docker/dockerfile:1.6

# =================================================================
# Definitive Production-Grade Multi-Stage Dockerfile
# =================================================================

# =================================================================
# Global Build Arguments
# =================================================================
ARG GO_VERSION=alpine3.22@sha256:daae04ebad0c21149979cd8e9db38f565ecefd8547cf4a591240dc1972cf1399
ARG TARGETPLATFORM=linux/amd64
# Metadata ARGs are passed in from the CI/CD pipeline for a single source of truth
ARG VERSION
ARG GIT_COMMIT
ARG BUILD_DATE
ARG SOURCE_URL="https://github.com/specialistvlad/burstgridgo"
ARG LICENSE="MIT"

# =================================================================
# Builder Stage
# =================================================================
FROM --platform=$BUILDPLATFORM golang:${GO_VERSION} AS builder

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
COPY ./examples/ ./examples/

# Run all checks: fmt, vet, lint, test
RUN go fmt ./... && \
    go vet ./... && \
    go test -v ./...

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
# Development Stage
# =================================================================
# This stage is intentionally locked to the build host's architecture ($BUILDPLATFORM)
# for performance, as emulating other platforms for development is often too slow.
FROM --platform=$BUILDPLATFORM golang:${GO_VERSION} AS dev

ARG UID=10002
RUN adduser -D -u ${UID} devuser

USER devuser
WORKDIR /app

ENV BGGO_APP_PORT=8080
EXPOSE 8080

# This stage pre-installs dependencies and tools. The source code is NOT
# copied into the image; it should be mounted as a volume during runtime.
# This is the standard pattern for enabling live-reloading.
COPY --chown=devuser:devuser go.mod go.sum ./

RUN go mod download && \
    go install github.com/air-verse/air@latest

COPY --chown=devuser:devuser .air.toml ./

ENTRYPOINT ["air"]

# =================================================================
# Production Stage
# =================================================================
ARG VERSION
ARG GIT_COMMIT
ARG BUILD_DATE
ARG SOURCE_URL
ARG LICENSE

# Use the 'cc' variant of distroless to include curl for a robust healthcheck.
FROM gcr.io/distroless/cc-debian12 AS prod

LABEL org.opencontainers.image.created=$BUILD_DATE \
    org.opencontainers.image.version="${VERSION}" \
    org.opencontainers.image.revision="${GIT_COMMIT}" \
    org.opencontainers.image.source="${SOURCE_URL}" \
    org.opencontainers.image.licenses="${LICENSE}"

ENV BGGO_APP_PORT=8080
EXPOSE 8080

COPY --from=builder /out/burstgridgo /burstgridgo
USER nonroot

# Uniform, robust healthcheck.
HEALTHCHECK --interval=15s --timeout=3s --start-period=5s --retries=3 \
    CMD ["curl", "--fail", "http://localhost:8080/health"]

ENTRYPOINT ["/burstgridgo"]
CMD ["--healthcheck-port 8080", "/grid/main.hcl"]
