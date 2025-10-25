# Makefile for building, testing, and running the burstgridgo application
IMAGE := burstgridgo
VERSION := $(shell git describe --tags --always)
BUILD_DATE := $(shell git log -1 --pretty=%cI)
GIT_COMMIT := $(shell git rev-parse --short HEAD)
HEALTHCHECK_PORT := 28080

# Set the default target to 'help'
.DEFAULT_GOAL := help

.PHONY: help build start dev test test-debug lint vet fmt coverage check docker-build-dev docker-dev docker-build-release

help: ## Show this help message.
	@echo "Usage: make [target] [options]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}' | \
		sed 's/|/\n                 /g'

# Allows running 'make start file.hcl' instead of 'make start ARGS=file.hcl'
ARGS ?= $(filter-out $@,$(MAKECMDGOALS))

build: ## Build the local application binary.
	@echo "Building application binary..."
	@mkdir -p .tmp
	go build -o .tmp/main ./cmd/cli

start: build ## Run the application locally (builds first).
	@echo "Starting application..."
	./.tmp/main $(ARGS)

dev: ## Run in dev mode with live-reloading (via air).
	@echo "Starting dev server with live-reloading (via 'go run github.com/air-verse/air')..."
	go run github.com/air-verse/air $(ARGS)

fmt: ## Format Go code (go fmt).
	@echo "Formatting code..."
	go fmt ./...

vet: ## Run Go vet linter.
	@echo "Running go vet..."
	go vet ./...

lint: ## Run the golangci-lint linter.
	@echo "Running golangci-lint..."
	golangci-lint run ./...

test: ## Run unit tests with race detection and coverage.
	@echo "Running tests..."
	go test -v -race -timeout 5s -coverprofile=coverage.out -coverpkg=./... ./...
	go tool cover -func=coverage.out

test-debug: ## Run unit tests with verbose debug logging.
	@echo "Running tests..."
	BGGO_TEST_LOGS=true go test -v -race -timeout 5s ./...

coverage: ## Open the test coverage report in your browser.
	@echo "Opening coverage report in browser..."
	go tool cover -html=coverage.out

check: fmt vet test ## Run all local checks (fmt, vet, test).
	@echo "Running full check (fmt + vet + test)..."

docker-build-dev: ## Build the development Docker image.
	@echo "Forcing a rebuild of the development image..."
	docker build --target dev -t $(IMAGE)-dev .

docker-dev: ## Run dev server in Docker with live-reloading.|   Options:|     grid=<path>   (Required) Path to HCL file.|     e="<vars>"    (Optional) Env vars to pass.|   Example:|     make docker-dev grid=examples/http_request.hcl e="API_KEY=secret"
	$(eval DOCKER_FLAGS := $(foreach pair,$(e),-e "$(pair)"))
	@docker image inspect $(IMAGE)-dev >/dev/null 2>&1 || make docker-build-dev
	@echo "Running dev container..."
	docker run --rm -it \
		-p $(HEALTHCHECK_PORT):8080 \
		-v $(PWD):/app \
		$(DOCKER_FLAGS) \
		-u "$(id -u):$(id -g)" \
		-w /app \
		$(IMAGE)-dev \
		$(grid)

docker-build-release: ## Build a release-ready Docker image.|   Creates tags for both version and 'latest'.
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