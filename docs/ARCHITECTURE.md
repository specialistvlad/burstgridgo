# Architecture
This document provides a deeper look into the internal architecture of `burstgridgo`. Understanding these concepts is essential for contributing new features or runners.

## Core Execution Flow
The application follows a simple but powerful execution flow, designed for concurrency and extensibility.

[DIAGRAM: Core_Execution_Flow]

1.  **HCL Parsing**: All `.hcl` files in the grid path are parsed into a flat list of module definitions.
2.  **DAG Construction**: A Directed Acyclic Graph (DAG) is built from the modules, representing the execution plan.
3.  **Concurrent Execution**: A worker pool executes the nodes of the DAG as their dependencies are met.
4.  **Telemetry & Reporting**: Results from the executor are fed to the TUI renderer and any configured telemetry exporters (OTLP, Prometheus).

## Anatomy of a Grid Run

### 1. HCL Parsing
The engine first parses all HCL files. A critical step in this phase is the expansion of *meta-arguments*. Any module with `count` or `for_each` is expanded into multiple, distinct module instances before the graph is built. This is how looping and conditional logic are handled.

### 2. DAG Construction
Once the final list of module instances is ready, the graph is constructed. Dependencies between nodes are determined in two ways:
* **Explicit Dependencies**: A module that uses the `depends_on` attribute.
* **Implicit Dependencies**: A module that references the output of another module in an expression (e.g., `${module.A.output}`).

Before execution, the graph is validated to ensure there are no circular dependencies.

### 3. Executor
The executor manages a pool of concurrent workers. It identifies all nodes in the graph with no dependencies and adds them to a work queue. When a worker finishes executing a module, it marks the node as complete, and the executor then identifies any new nodes whose dependencies are now fully met, adding them to the queue.

## Building a Custom Runner
`burstgridgo` is a framework, and its primary extension point is the `Runner` interface.

### The `Runner` Interface
To create a new runner, you must implement the `engine.Runner` interface in Go.
```go
// From: ./internal/engine/module.go

// Runner defines the interface that all modules must implement to be executable.
type Runner interface {
	// Run now accepts an EvalContext to resolve inputs and returns a cty.Value for its output.
	Run(m Module, ctx *hcl.EvalContext) (cty.Value, error)
}
```
* **`m Module`**: Provides access to the HCL configuration for this module instance.
* **`ctx *hcl.EvalContext`**: Used to evaluate HCL expressions and access outputs from dependencies.

The function returns a `cty.Value` as its output and an `error` if it fails.

### Registration
You must register your runner in an `init()` function within your runner's package to make it available to the HCL engine.

### Managing Dependencies
If your runner requires third-party Go packages, add the `import` statement to your code and run the following command from the project root. This will add the dependency to the project's `go.mod` file.
```sh
go mod tidy
```

### Designing Complex, Stateful Runners
For complex protocols like Socket.IO or gRPC streams, a single runner can act as a mini-orchestrator by defining its own internal Domain-Specific Language (DSL) using nested HCL blocks.

The runner's `Run` method is responsible for parsing these nested blocks and executing them in the correct sequence, managing its own internal state (e.g., an active connection).
```go
// From: ./modules/socketio/module.go

// Config defines the HCL structure for this module.
type Config struct {
	URL                string    `hcl:"url"`
	Namespace          string    `hcl:"namespace,optional"`
	OnEvent            string    `hcl:"on_event"`
	EmitEvent          string    `hcl:"emit_event,optional"`
	EmitData           cty.Value `hcl:"emit_data,optional"`
	Timeout            string    `hcl:"timeout,optional"`
	InsecureSkipVerify bool      `hcl:"insecure_skip_verify,optional"`
}
```

## Plugin Ecosystem
There are two primary ways to add a custom runner to the project.

* **In-Project Method**: Clone the main `burstgridgo` repository and add your runner code directly to the `./modules` directory. This is the simplest method.
* **Advanced Method (The Vision)**: In the future, a standalone `bggo-builder` tool will allow you to compile a custom `burstgridgo` binary with third-party runners without cloning the main project.