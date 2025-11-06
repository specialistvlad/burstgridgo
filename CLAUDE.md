# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

BurstGridGo is a declarative automation and orchestration engine built in Go. It parses HCL configuration files to define workflows as directed acyclic graphs (DAGs) and executes them with support for parallelism, dependency management, and modularity.

**Status:** MVP Stage 1 - API and architecture are not yet stable and subject to breaking changes.

## Common Commands

### Building and Running
- `make build` - Build the CLI binary to `.tmp/main`
- `make run <file.hcl>` - Build and run with an HCL file (e.g., `make run ./examples/http_request.hcl`)
- `make dev-watch <file.hcl>` - Run in development mode with live-reloading via air

### Testing
- `make test` - Run all tests with race detection and coverage
- `make test-watch` - Run tests continuously on file changes (useful during development)
- `make test-debug` - Run tests with verbose debug logging (set `BGGO_TEST_LOGS=true`)
- `make coverage` - Open HTML coverage report in browser (run `make test` first)

### Code Quality
- `make fmt` - Format all Go code
- `make vet` - Run go vet linter
- `make lint` - Run golangci-lint
- `make check` - Run all checks (fmt + vet + test) before committing

### Docker
- `make docker-dev grid=<path>` - Run dev container with live-reloading
  - Example: `make docker-dev grid=examples/http_request.hcl e="API_KEY=secret"`

### Running Single Tests
```bash
go test -v -run TestName ./path/to/package
go test -v -run TestName ./...  # Run across all packages
```

## Architecture Overview

BurstGridGo follows a **declarative execution pipeline** that transforms HCL configuration into an executable DAG. The architecture is organized into distinct phases:

### 1. Parsing Phase (HCL → Model)
- **Entry Point:** `cmd/cli/main.go` - CLI entry point, parses args via `internal/cli`
- **HCL Parsing:** `internal/bggohcl` - Parses `.hcl` files using HashiCorp's HCL library
- **Model Construction:** `internal/model` - Creates strongly-typed Go structs representing the configuration
  - `Grid`: Root container for the entire workspace
  - `Runner`: Reusable task definitions (templates)
  - `Step`: Instances of runners with specific arguments (nodes in the DAG)
  - `FSInfo`: File metadata for error reporting

### 2. Expression Evaluation
- **Expression Engine:** `internal/bggoexpr` - Evaluates HCL expressions using go-cty
- Resolves references like `resource.http_client.shared` and `step.http_request.first.output`
- Supports templating, locals, variables, and cross-step dependencies

### 3. Graph Construction
- **Builder:** `internal/builder` - Transforms the model into an executable DAG
- **Graph:** `internal/graph` - High-level stateful representation of the DAG during execution
- **Topology Store:** `internal/inmemorytopology` - Manages DAG structure and node relationships
- **Node Store:** `internal/inmemorystore` - Tracks node state during execution

### 4. Execution Phase
- **Session:** `internal/session` & `internal/localsession` - Abstracts execution environment (local vs distributed)
- **Executor:** `internal/executor` & `internal/localexecutor` - Orchestrates DAG execution
- **Scheduler:** `internal/scheduler` - Determines which nodes are ready to execute based on dependencies
- **Task:** `internal/task` - Represents a fully-resolved, runnable node
- **Handlers/Registry:** `internal/handlers` & `internal/registry` - Maps runner types to their implementations

### 5. Modules
- **Location:** `modules/` - Pluggable runner implementations (e.g., `modules/print`)
- Modules define the actual execution logic for different runner types
- Currently minimal; will expand into a modular marketplace ecosystem

### Key Design Patterns
1. **Declarative First:** Users define *what* to run, not *how*. The engine figures out execution order from dependencies.
2. **Layered Architecture:** Clean separation between parsing, validation, graph building, and execution.
3. **Type-Safe Model:** HCL is converted into strongly-typed Go structs before any evaluation happens.
4. **Expression Isolation:** HCL expressions are only evaluated during graph construction, not during execution.
5. **Store Pattern:** State and topology are managed via separate store interfaces for flexibility.

## Key Concepts

### Node Addressing
The `internal/nodeid` package provides a robust addressing system:
- Format: `<type>.<name>` or `<type>.<name>[index]` for loops
- Examples: `step.http_request.first`, `resource.http_client.shared`
- Used throughout the system for referencing nodes and resolving dependencies

### Dependency Management
- Steps declare dependencies via `depends_on` attribute
- The scheduler analyzes the graph to determine execution order
- Supports fan-in and fan-out patterns (see `examples/http_count_static_fan_in.hcl`)

### Context Logging
The `internal/ctxlog` package provides context-aware structured logging using `slog`. Always retrieve loggers from context rather than creating global instances.

## Testing Strategy

- **Unit Tests:** Most packages have `_test.go` files colocated with implementation
- **Integration Tests:** `internal/integrationtests/` contains end-to-end tests that parse and execute full HCL files
- **Test Utilities:** `internal/testutil/` provides harnesses for parsing and testing runners/steps
- **Coverage:** Project maintains test coverage tracking via codecov

## Project Structure

```
├── cmd/cli/              # CLI entry point
├── internal/
│   ├── app/              # Application initialization and lifecycle
│   ├── cli/              # CLI argument parsing
│   ├── model/            # Core data structures (Grid, Runner, Step)
│   ├── bggohcl/          # HCL parsing utilities
│   ├── bggoexpr/         # Expression evaluation engine
│   ├── builder/          # DAG construction
│   ├── graph/            # Stateful DAG representation
│   ├── scheduler/        # Node scheduling logic
│   ├── executor/         # Execution orchestration interfaces
│   ├── localexecutor/    # Local execution implementation
│   ├── session/          # Session abstraction
│   ├── localsession/     # Local session implementation
│   ├── registry/         # Runner type registry
│   ├── handlers/         # Runner implementation handlers
│   ├── task/             # Runnable task representation
│   ├── node/             # Node abstractions
│   ├── nodeid/           # Node addressing system
│   ├── inmemorystore/    # In-memory node state storage
│   ├── inmemorytopology/ # In-memory topology storage
│   ├── integrationtests/ # End-to-end tests
│   └── testutil/         # Testing utilities
├── modules/              # Pluggable runner implementations
└── examples/             # Example HCL configuration files
```

## Development Workflow

1. **Make Changes:** Edit code in your preferred editor
2. **Run Tests:** Use `make test-watch` for rapid feedback during development
3. **Run Example:** Test with `make dev-watch ./examples/http_request.hcl`
4. **Debug:** Use `make test-debug` to enable verbose logging
5. **Pre-Commit:** Run `make check` to ensure fmt, vet, and tests pass

## Important Notes

- **Go Version:** Requires Go 1.25
- **Dependencies:** Uses HashiCorp HCL v2 and go-cty for configuration parsing
- **Logging:** Always use `ctxlog.FromContext(ctx)` rather than global loggers
- **Errors:** Provide clear error messages with file/line context from `FSInfo`
- **Panics:** CLI has panic recovery in `cmd/cli/main.go` - do not suppress or hide panics elsewhere

## Examples

Example HCL files in `examples/` demonstrate:
- `http_request.hcl` - Basic HTTP requests with dependencies and resource sharing
- `http_count_static_fan_in.hcl` - Parallel execution with fan-in pattern
- `env_*.hcl` - Environment variable handling
- `socketio_ping_pong.hcl` - WebSocket example

Use these as references when implementing new features or testing changes.
