# Makefile for building and running the burstgridgo application using Docker
IMAGE := burstgridgo
VERSION := $(shell git describe --tags --always)
BUILD_DATE := $(shell git log -1 --pretty=%cI)
GIT_COMMIT := $(shell git rev-parse --short HEAD)
HEALTHCHECK_PORT := 28080

# The user must now provide the grid path.
grid ?=

ARCH := $(shell uname -m)
ifeq ($(ARCH),x86_64)
  PLATFORM ?= linux/amd64
else ifneq (,$(filter $(ARCH),aarch64 arm64))
  PLATFORM ?= linux/arm64
endif

# Set the default target to 'help'
.DEFAULT_GOAL := help

.PHONY: help build prod dev

help: ## Show this help message.
	@echo "Usage: make [target] [options]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}' | \
		sed 's/|/\n                 /g'

build: ## Build a production-ready Docker image.|   Creates tags for both version and 'latest'.
	docker buildx build \
	  --target prod \
	  --platform=$(PLATFORM) \
	  --load \
	  --build-arg VERSION=$(VERSION) \
	  --build-arg GIT_COMMIT=$(GIT_COMMIT) \
	  --build-arg BUILD_DATE=$(BUILD_DATE) \
	  -t $(IMAGE):$(VERSION) \
	  -t $(IMAGE):latest \
	  .

prod: build ## Run the latest production image.|   Note: This default command does not mount a grid.
	docker run --rm -p $(HEALTHCHECK_PORT):8080 $(IMAGE):$(VERSION)

dev: ## Run dev container with live-reloading.|   Options:|     grid=<path>   (Required) Path to HCL file.|     e="<vars>"    (Optional) Env vars to pass.|   Example:|     make dev grid=examples/dev.hcl e="API_KEY=secret"
	$(eval DOCKER_FLAGS := $(foreach pair,$(e),-e "$(pair)"))
	docker build \
	  --target dev \
	  -t $(IMAGE)-dev .
	@echo "Running dev container..."
	docker run --rm -it \
	-p $(HEALTHCHECK_PORT):8080 \
	-v $(PWD):/app \
	$(DOCKER_FLAGS) \
	-u "$(id -u):$(id -g)" \
	-w /app \
	$(IMAGE)-dev \
	$(grid)

# Add this with your other phony targets
.PHONY: help build prod dev test

# Add this target, for example after the 'help' target
test: ## Run all tests with race detection.
	@echo "Running tests..."
	go test -v -race ./...