# ADR-002: Core Engine Internal Refactoring
- **Status**: Implemented
- **Author**: Vladyslav Kazantsev
- **Date**: 2025-07-26

---
## 1. Context

The current implementation of the core engine, particularly within the `internal/dag` package, has proven the validity of our architecture. However, as features have been added, several key files have grown significantly in size and complexity.

Specifically, **`internal/dag/executor.go`** now manages the worker pool, node execution for both steps and resources, HCL evaluation context construction, dependency injection for the `uses` block, and resource cleanup logic. Similarly, the main application entrypoint at **`cmd/cli/main.go`** contains a large block of procedural code for discovering, parsing, and merging HCL grid files.

This concentration of logic in a few large files increases cognitive overhead, making the system harder to maintain and onboard new contributors. The goal of this ADR is to refactor these internals for better code health without altering the established architecture or external behavior.

---
## 2. Decision

We will undertake a pure internal refactoring to decompose large files into smaller, more focused units. This is a non-functional change that will not alter the engine's public API, behavior, or core architectural principles.

### Decision 1: Decompose the Executor
The **`internal/dag/executor.go`** file will be broken down into multiple files within the same `dag` package, each with a single, clear responsibility.

* **`executor.go`**: This will remain the primary file for the `Executor` struct. It will contain the high-level orchestration logic: the `Run()` method, the `worker` pool management, and channel communication. It will delegate specific tasks to the new supporting files.
* **`node_runner.go`**: This new file will contain the logic for executing a single graph node. It will house the `executeStepNode` and `executeResourceNode` functions, which are responsible for finding handlers, decoding arguments, and invoking the appropriate Go functions.
* **`context_builder.go`**: This new file will be responsible for building the HCL evaluation context. It will contain the `buildEvalContext` function, which makes `step.x.output` and other variables available for HCL interpolation.
* **`deps_builder.go`**: This new file will manage the dependency injection for the `uses` block. It will contain the `buildDepsStruct` function, which populates a step's `Deps` struct with live resource instances.
* **`cleanup.go`**: This new file will centralize the resource lifecycle management logic from the executor's perspective, including the `pushCleanup`, `executeCleanupStack`, and efficient `destroyResource` functions.

### Decision 2: Encapsulate Grid Loading Logic
A new high-level function will be introduced in the `engine` package. This function will encapsulate the entire grid loading process currently handled in `cmd/cli/main.go`, including path resolution, discovery of HCL files, parsing and decoding, merging multiple files into a single configuration, and injecting the default `help` step when necessary. This moves the configuration loading and preparation responsibility into the `engine` package, where it logically belongs.

### Decision 3: Simplify the Main Application Entrypoint
As a direct result of Decision 2, the `main` function in `cmd/cli/main.go` will become significantly cleaner and more declarative. The large, complex block of file handling and configuration merging logic will be replaced by a single, descriptive call to the new function in the `engine` package. The entrypoint's responsibility will be reduced to high-level orchestration: parsing CLI options, setting up the logger, loading the grid configuration, building the graph, and running the executor.

---
## 3. Consequences

#### Pros 👍
* **Improved Maintainability**: Smaller files with a single responsibility are easier to read, understand, and modify safely.
* **Reduced Cognitive Load**: Developers can focus on a specific piece of functionality (e.g., context building) without needing to parse the entire executor's logic.
* **Zero Architectural Change**: This is a pure code health refactoring. All existing concepts like runners, assets, the DAG, and handler registration remain unchanged.
* **No Impact on Tests**: Because this is a refactoring of private functions and internal file structure, the existing integration tests will pass without modification, confirming that the public behavior of the executor is preserved.

#### Cons 👎
* **Increased File Count**: The number of files in the `internal/dag` directory will increase. This is a deliberate and positive trade-off for improved clarity.
* **One-Time Refactoring Effort**: There is a development cost to carefully move the code and ensure all connections are preserved.