# Makefile for building and running the burstgridgo application using Docker
IMAGE := burstgridgo
VERSION := $(shell git describe --tags --always)
BUILD_DATE := $(shell git log -1 --pretty=%cI)
GIT_COMMIT := $(shell git rev-parse --short HEAD)
HEALTHCHECK_PORT := 28080

# The user must now provide the grid path.
grid ?=

# Set the default target to 'help'
.DEFAULT_GOAL := help

# Add this with your other phony targets
.PHONY: help build build-dev prod dev test test-debug lint vet fmt check coverage

ARCH := $(shell uname -m)
ifeq ($(ARCH),x86_64)
  PLATFORM ?= linux/amd64
else ifneq (,$(filter $(ARCH),aarch64 arm64))
  PLATFORM ?= linux/arm64
endif


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

build-dev: ## Force a rebuild of the development Docker image.
	@echo "Forcing a rebuild of the development image..."
	docker build --target dev -t $(IMAGE)-dev .
	
dev: ## Run dev container with live-reloading.|   Options:|     grid=<path>   (Required) Path to HCL file.|     e="<vars>"    (Optional) Env vars to pass.|   Example:|     make dev grid=examples/http_request.hcl e="API_KEY=secret"
	$(eval DOCKER_FLAGS := $(foreach pair,$(e),-e "$(pair)"))
	@docker image inspect $(IMAGE)-dev >/dev/null 2>&1 || make build-dev
	@echo "Running dev container..."
	docker run --rm -it \
		-p $(HEALTHCHECK_PORT):8080 \
		-v $(PWD):/app \
		$(DOCKER_FLAGS) \
		-u "$(id -u):$(id -g)" \
		-w /app \
		$(IMAGE)-dev \
		$(grid)


fmt: ## Format code with go fmt
	@echo "Formatting code..."
	go fmt ./...

vet: ## Run go vet
	@echo "Running go vet..."
	go vet ./...

lint: ## Run optional golangci-lint
	@echo "Running golangci-lint..."
	golangci-lint run ./...

test: ## Run unit tests
	@echo "Running tests..."
	go test -v -race -timeout 5s -coverprofile=coverage.out -coverpkg=./... ./...
	go tool cover -func=coverage.out

test-debug: ## Run unit tests with logs enabled in debug mode BGGO_TEST_LOGS=true
	@echo "Running tests..."
	BGGO_TEST_LOGS=true go test -v -race -timeout 5s ./...

coverage: ## Display a test coverage report
	@echo "Opening coverage report in browser..."
	go tool cover -html=coverage.out

check: fmt vet test ## Run all local validations (CI-safe)
	@echo "Running full check (fmt + vet + test)..."