# syntax=docker/dockerfile:1.6

# =================================================================
# Global Build Arguments
# =================================================================
ARG GO_VERSION=1.24.5
ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG BINARY_NAME=burstgridgo
ARG BUILD_DATE
ARG VERSION

# =================================================================
# Base Stage (Common Dependency Layer)
# =================================================================
# Downloads dependencies to be reused by other stages.
# $BUILDPLATFORM ensures this runs natively on the builder machine for performance.
FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine AS base
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

# =================================================================
# Builder Stage (Compiles the Application)
# =================================================================
# Builds the static, cross-platform binary from the source code.
FROM base AS builder
COPY ./cmd/ ./cmd/
COPY ./internal/ ./internal/
COPY ./modules/ ./modules/

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -ldflags="-s -w -X 'main.version=${VERSION}' -X 'main.buildDate=${BUILD_DATE}'" \
    -o /out/${BINARY_NAME} \
    ./cmd/cli

# =================================================================
# Production Stage (Final, Minimal Image)
# =================================================================
# This is the final, secure image for production. It uses the target
# platform specified in the `docker buildx build` command.
FROM --platform=${TARGETOS}/${TARGETARCH} gcr.io/distroless/static-debian12:nonroot AS prod
LABEL org.opencontainers.image.created=$BUILD_DATE \
      org.opencontainers.image.version=$VERSION \
      org.opencontainers.image.source="https://github.com/vlad/burstgridgo"

COPY --from=builder /out/${BINARY_NAME} /${BINARY_NAME}
HEALTHCHECK CMD ["/burstgridgo", "--healthcheck"]
ENTRYPOINT ["/burstgridgo"]

# =================================================================
# Development Stage (For Live Reloading)
# =================================================================
# This stage is for local development with live-reloading.
# It runs on the same platform as your build machine.
FROM --platform=$BUILDPLATFORM base AS dev
RUN go install github.com/cosmtrek/air@latest
USER nobody
# The source code should be mounted as a volume into /app during development.
CMD ["air"]

# =================================================================
# Debug Stage (Includes Shell and Tools)
# =================================================================
# This stage includes the compiled binary plus common debugging tools.
# It uses the target platform to allow debugging the exact binary.
FROM --platform=${TARGETOS}/${TARGETARCH} alpine:latest AS debug
COPY --from=builder /out/${BINARY_NAME} /${BINARY_NAME}
RUN apk add --no-cache bash curl strace lsof
ENTRYPOINT ["/burstgridgo"]