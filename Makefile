# Makefile for building and running the burstgridgo application using Docker
IMAGE := burstgridgo
VERSION := $(shell git describe --tags --always)
# Use the last commit's timestamp for a stable, cache-friendly build date
BUILD_DATE := $(shell git log -1 --pretty=%cI)
GIT_COMMIT := $(shell git rev-parse --short HEAD)

ARCH := $(shell uname -m)
ifeq ($(ARCH),x86_64)
  PLATFORM ?= linux/amd64
else ifneq (,$(filter $(ARCH),aarch64 arm64))
  PLATFORM ?= linux/arm64
endif

.PHONY: build
build:
	docker buildx build \
	  --target prod \
	  --platform=$(PLATFORM) \
	  --load \
	  --build-arg VERSION=$(VERSION) \
	  --build-arg GIT_COMMIT=$(GIT_COMMIT) \
	  --build-arg BUILD_DATE=$(BUILD_DATE) \
	  -t $(IMAGE):$(VERSION) .

.PHONY: prod
prod: build
	@docker image inspect $(IMAGE):$(VERSION) >/dev/null 2>&1 || \
		make build
	docker run --rm -p 8080:8080 $(IMAGE):$(VERSION)

.PHONY: dev
dev:
	docker build \
	  --target dev \
	  -t $(IMAGE)-dev .
	docker run --rm -it \
	-p 8080:8080 \
	-v $(PWD):/app \
	-w /app \
	$(IMAGE)-dev
