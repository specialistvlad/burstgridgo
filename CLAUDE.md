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

### Testing Philosophy: Integration Tests First

**This project prioritizes integration tests over unit tests.** The rationale:

1. **Resilience to Refactoring:** Internal implementation details change frequently. Integration tests remain valid even when internal packages are restructured or refactored.
2. **Reduced Maintenance Burden:** When code changes, you don't need to rewrite dozens of unit tests. Integration tests focus on behavior, not implementation.
3. **End-to-End Coverage:** Integration tests verify the entire solution works together, catching issues that unit tests might miss.
4. **Confidence in Refactoring:** With solid integration tests, you can aggressively refactor internals (like ADR-002, ADR-013) without fear of breaking behavior.

### Test Hierarchy

**Priority 1: Integration Tests** (`internal/integrationtests/`)
- End-to-end tests that parse real HCL files and verify complete workflows
- Test actual user-facing behavior and contracts
- Currently **67.5% coverage** - the highest in the project
- Examples: `hcl_runner_basic_test.go`, `hcl_step_deps_test.go`, `app_loader_test.go`
- **Always write integration tests first** when adding new features

**Priority 2: Critical Unit Tests**
- Only for complex algorithms or non-obvious logic
- Data structures with intricate behavior (e.g., concurrent stores, graph algorithms)
- Examples: `internal/inmemorystore`, `internal/nodeid`, `internal/bggoexpr`

**Priority 3: Test Utilities** (`internal/testutil/`)
- Harnesses for common testing patterns
- Helpers for parsing and constructing test fixtures
- Shared test infrastructure

### When to Write Unit Tests

Write unit tests when:
- Testing complex algorithms in isolation (e.g., node address parsing, expression extraction)
- Verifying thread-safety and concurrent behavior
- Testing error conditions that are hard to trigger via integration tests
- Performance-critical code where you need precise benchmarks

**Do NOT write unit tests for:**
- Simple data structure methods (getters/setters)
- Code that just wires dependencies together
- Logic that is fully covered by integration tests
- Internal implementation details that might change

### Test Execution

```bash
# Run all tests (integration + unit)
make test

# Watch tests during development
make test-watch

# Debug with verbose logging
make test-debug  # Sets BGGO_TEST_LOGS=true
```

### Coverage Philosophy

- **Low unit test coverage is acceptable** if integration tests cover the behavior
- Packages with 0% coverage are often:
  - Placeholder implementations (executor, scheduler)
  - Simple interfaces or models
  - Code tested at integration level
- **Integration test coverage (67.5%) is the key metric**
- Race detector enabled (`-race` flag) on all test runs

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

---

## Architecture Decision Records (ADRs)

This project uses Architecture Decision Records to document significant architectural and design decisions. ADRs are located in `docs/ADR/`.

### ADR Structure
Each ADR follows a consistent format:
- **Header:** Number, Title, Status (Draft/Accepted/Implemented), Author, Date
- **Context:** Problem statement and why the decision is needed
- **Decision Drivers:** (Optional) Factors influencing the decision
- **Decision/Decision Outcome:** What is being decided with specific implementation details
- **Consequences:** Positive and negative impacts
- **Implementation Plan:** (Optional) Step-by-step implementation approach
- **Validation and Testing:** (Optional) How to verify the implementation

### Key ADRs to Reference
- **ADR-001:** State Management - Resource lifecycle and dependency injection via `uses` block
- **ADR-002:** Internal Refactoring - Pattern for decomposing large files into focused units
- **ADR-013:** DAG Topology Extraction - Separation of concerns between business logic and graph operations
- **ADR-014:** Structured Node Identifiers - Node addressing system architecture
- **ADR-xxx-executor-refactoring:** Splitting executor into generic DAG engine and HCL-specific runner
- **ADR-xxx-fan-in:** Draft for dynamic `count` and `for_each` aggregation

### When to Create an ADR
Create an ADR when:
1. Making significant architectural decisions (new packages, major refactoring)
2. Introducing new concepts or patterns (e.g., stateful resources, type systems)
3. Changing public APIs or contracts
4. Decisions that affect multiple packages or have long-term consequences

### What Does NOT Need an ADR
**ADRs are for architecture, not routine development.** The following do NOT require an ADR:
- ❌ Adding new modules (e.g., `modules/redis`, `modules/postgres`)
- ❌ Bug fixes and error handling improvements
- ❌ Documentation updates
- ❌ Adding tests or test utilities
- ❌ Small feature additions to existing components
- ❌ Performance optimizations that don't change architecture
- ❌ Code formatting and linting fixes
- ❌ Adding new fields to existing structs (unless changing contracts)

**Rule of thumb:** If it's implementing an existing pattern or fixing/enhancing something without introducing new concepts, you don't need an ADR. Just write good code and tests.

### ADR Naming Convention
- **Implemented:** `ADR-NNN-descriptive-name.md` (numbered sequentially)
- **Draft/In Progress:** `ADR-xxx-descriptive-name.md` (use `xxx` prefix)
- Update status header as: Draft → Accepted → Implemented

### ADR Writing Principles
- Focus on **why** not just **what** - explain the reasoning and trade-offs
- Document both positive and negative consequences
- Reference specific package and file names
- Include implementation verification strategy (tests, validation)
- Link to related ADRs when there are dependencies
- Add implementation notes when status changes to "Implemented"

---

## Current Project State & Next Steps

### ✅ What's Working
- **HCL Parsing Pipeline:** Successfully parses `.hcl` files into strongly-typed Go models (`internal/model`)
- **Model Layer:** Grid, Runner, Step structures complete and well-documented
- **CLI & App Infrastructure:** Argument parsing, logging, configuration functional
- **Module System Foundation:** Handler registration works (`modules/print` implemented)
- **Integration Tests:** 9 of 11 integration test suites passing (2 have skipped tests)

### 🚧 Critical Gap: Execution Pipeline Not Implemented
The entire execution layer is currently stubbed out with placeholder implementations:

**Location:** `internal/app/app.go:57-77` - execution code is commented out

**Affected Components:**
1. **Scheduler** (`internal/scheduler/scheduler.go`)
   - Returns empty channel, no logic to determine ready nodes
   - Needs: Analyze graph state, find nodes with satisfied dependencies, emit via channel

2. **Builder** (`internal/builder/builder.go`)
   - Returns empty task with no input resolution
   - Needs: Resolve HCL expressions, evaluate `uses` and `arguments`, produce fully-resolved Task

3. **Executor** (`internal/localexecutor/executor.go`)
   - Just logs "placeholder", doesn't execute anything
   - Needs: Pull tasks from scheduler, dispatch to handlers, manage goroutine pool, update node state

4. **Graph Construction**
   - Model → Graph transformation not fully implemented
   - Needs: Create nodes from Steps/Resources, build topology from `depends_on`

### Strategic Next Steps - Three Paths

#### **Option A: Complete Execution Core** 🎯 **(HIGHEST PRIORITY)**
**Goal:** Make the engine actually run workflows end-to-end

**Why First:**
- Without execution, modules and features can't be properly tested
- Validates the beautifully designed 5-phase architecture
- Enables contributors to build modules with confidence
- Moves from "parses config" to "executes workflows" - true MVP completion

**Implementation Phases:**
1. Graph Builder - Transform `model.Grid` to executable DAG
2. Scheduler - Analyze dependencies, emit ready nodes
3. Builder - Resolve expressions, produce runnable tasks
4. Executor - Dispatch tasks to handlers, manage concurrency
5. Integration - Uncomment execution code in `app.go`, create end-to-end tests

**Quick Win Approach:**
- Start minimal (sequential execution, hardcoded values)
- Get one example running end-to-end
- Iterate: add concurrency → expression eval → error handling

#### **Option B: Expand Module Ecosystem** 🧩
**Goal:** Build out modular runner marketplace

**Focus:** HTTP enhancements (#54, #55), filesystem utils (#52), databases (Redis #60, Postgres #61, Mongo #62), messaging (RabbitMQ #63, Kafka #64)

**Challenge:** Cannot be fully tested without Option A execution core

#### **Option C: Feature Completeness** ✨
**Goal:** Implement missing HCL language features

**Tasks:** Variables support (#50), CLI flags (#51), splat operator (#49), complete skipped integration tests

**Impact:** Increases HCL expressiveness but doesn't enable actual workflow execution

### TODOs in Codebase
Key TODOs found in source (via grep):
- `internal/model/placement.go:13` - Update documentation
- `internal/testutil/harness.go:115` - Fix harness issue (low priority)
- Multiple skipped integration tests in `internal/integrationtests/hcl_*_test.go`
- `internal/model/error_handling.go:13` - Describe error handling
- `internal/model/observability.go:13` - Describe observability details

### GitHub Issues Context
67+ open issues, primarily module requests (issues #48-67). These represent the future module marketplace but require execution core to be useful.

---

## Working with This Codebase

### Before Starting New Work
1. **Check ADRs:** Review `docs/ADR/` for architectural decisions related to your work
2. **Run Tests:** Ensure `make test` passes before making changes
3. **Review Integration Tests:** Check `internal/integrationtests/` for examples of how features should work
4. **Check TODOs:** Search for related TODOs or FIXMEs in the area you're modifying

### Making Architectural Changes
1. **Document First:** Create a draft ADR (`ADR-xxx-name.md`) before major changes
2. **Discuss Trade-offs:** Explicitly document positive and negative consequences
3. **Reference Patterns:** Look at ADR-002 for refactoring patterns, ADR-013 for separation concerns
4. **Test Strategy:** Define how the change will be verified before implementing

### Code Quality Standards
- **Separation of Concerns:** Each package should have a single, clear responsibility (see ADR-002, ADR-013)
- **Interface-Driven:** Use interfaces for dependency injection and testability
- **Context-Aware:** Always pass `context.Context` and use `ctxlog.FromContext(ctx)` for logging
- **Test Coverage:** Aim for high coverage, especially for core DAG/graph logic
- **Documentation:** Use package-level `doc.go` files to explain purpose and responsibilities
