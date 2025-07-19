# syntax=docker/dockerfile:1.6

ARG GO_VERSION=1.24.5
ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG BINARY_NAME=burstgridgo
ARG BUILD_DATE
ARG VERSION

# 1. Build stage
FROM --platform=${TARGETOS}/${TARGETARCH} golang:${GO_VERSION}-bookworm AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go install github.com/cosmtrek/air@latest && go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o /out/${BINARY_NAME} ./cmd/cli

# 2. Final stage - production
FROM --platform=${TARGETOS}/${TARGETARCH} gcr.io/distroless/static-debian12:nonroot AS prod
LABEL org.opencontainers.image.created=$BUILD_DATE \
      org.opencontainers.image.version=$VERSION \
      org.opencontainers.image.source="https://github.com/vlad/burstgridgo"
COPY --from=builder /out/${BINARY_NAME} /${BINARY_NAME}
USER nonroot:nonroot
HEALTHCHECK CMD ["/burstgridgo", "--healthcheck"]
ENTRYPOINT ["/burstgridgo"]

# 3. Final stage - development
FROM --platform=${TARGETOS}/${TARGETARCH} golang:${GO_VERSION}-bookworm AS dev
WORKDIR /app
COPY go.mod go.sum ./
RUN go install github.com/cosmtrek/air@latest && go mod download
COPY . .
CMD ["air"]

# 4. Debug image stage
FROM alpine AS debug
COPY --from=builder /out/${BINARY_NAME} /${BINARY_NAME}
RUN apk add --no-cache bash curl
ENTRYPOINT ["/${BINARY_NAME}"]
