# 🧭 Project Roadmap
This document outlines the development roadmap for `burstgridgo`. Our vision is to create the best tool for defining complex load tests as code and turning the results into actionable insights. This plan incorporates our recent architectural discussions to evolve the platform's foundation.

This roadmap is a living document. Priorities may shift based on community feedback and technical needs.

---

### Completed ✅
* **Core Engine v1**: Foundational DAG-based executor with HCL parsing.
* **Extensible Runner Architecture**: The core `engine.Runner` interface and dynamic registration system.
* **Initial Core Runners**: Shipped runners for `http-request`, `socketio`, `s3`, `print`, and `env_vars`.
* **Containerized DX**: Multi-stage `Dockerfile` and `Makefile` for reproducible development and production builds.

---

### Next Up: Foundational Refactor & Critical Fixes 🎯
This cycle focuses on a significant architectural evolution to align with our discussion, alongside fixing critical bugs in the build and execution pipeline. These changes will provide a stable, scalable, and developer-friendly foundation for all future work.

#### 1. Core Architecture Evolution (Implement Our Design)
This is the highest priority. We will refactor the core engine and runner interface to match the robust design we established.

* HCL files define a runner, not a runner.go:
    Final Architecture Plan
This architecture creates a robust, type-safe contract between the HCL configuration and the Go code. It separates the definition of a runner from its execution, making the system more modular and self-documenting.

1. The Runner Definition (The runner Block)

A runner's public API is now defined in HCL files located within its module directory (e.g., modules/http_request/).

Discovery: The application will find these definitions by recursively scanning the modules/ directory at startup.

Structure: The definition is wrapped in a runner "type" {} block.

Example Definition (modules/http_request/manifest.hcl):

Terraform
runner "http_request" {
  description = "Executes a simple HTTP request."

  input "url" {
    type        = string
    description = "The URL to send the request to."
  }

  input "method" {
    type        = string
    description = "The HTTP method to use."
    optional    = true
    default     = "GET"
  }

  output "status_code" {
    type        = number
    description = "The HTTP status code of the response."
  }

  lifecycle {
    on_run = "OnRunHttpRequest"
  }
}
2. The Execution Instance (The step Block)

A user executes a runner in their grid file by using a step block. This creates an instance of a defined runner.

Structure: A step "type" "name" {} block is used to call a runner.

arguments {} Block: Provides the actual values for the inputs defined in the runner manifest.

Dependencies: Uses depends_on and HCL interpolation (${step.other_step.output.field}) to build the DAG.

Example Execution (my_test.hcl):

Terraform
step "http_request" "get_homepage" {
  arguments {
    url = "https://example.com"
  }
}
3. The Go Handler Implementation

The Go code acts as a library of handler functions that are explicitly registered with the engine.

Registration: Each handler is registered by its string name in a global map using Go's init() function (e.g., engine.Register("OnRunHttpRequest", OnRunHttpRequest)).

Stateful Signatures: The handlers use a standard signature that passes state between lifecycle events.

on_start creates a state object: func(...) (*State, error)

on_run receives that state: func(state *State, ...) (*Output, error)

on_end also receives the state for cleanup: func(state *State) error

Typed I/O: Handlers use native Go structs for inputs and outputs. Output structs must use cty:"snake_case_name" tags to expose their fields to HCL.

4. The Executor's Role (The Bridge)

The executor connects the HCL configuration to the Go logic.

Validation: Before running, the executor performs a pre-flight check, validating the arguments in each step against the input schema of the corresponding runner. This catches errors early.

State Management: It calls the on_start handler, captures the returned *State object, and passes it to on_run and on_end for that specific step instance.

Data Flow:

It decodes the HCL arguments into the typed Go *Input struct for the handler.

After the handler returns a Go *Output struct, the executor uses cty.ToValue() to convert it into a cty.Value. This makes the output available to other steps in the DAG.


* **Implement Shared resources between runners**:
    * **Problem**: The `http-request` and `s3` runners create a new `http.Client` for every single execution. This is inefficient as it prevents TCP connection reuse (Keep-Alive).
    * **Fix**: Refactor these runners to use a shared, package-level `http.Client` instance, improving performance for high-throughput tests.

---

### Future Vision & Backlog 💡
This is a list of features and less critical issues that are planned but not yet scheduled.

#### Pillar: Foundation & Developer Experience (DX)
* **Comprehensive Test Coverage**: Add robust, table-driven test suites for the `dag` package and core engine logic, covering complex dependency scenarios and error conditions.
* **Standardized Error Handling & Fail-Fast Mode**: Introduce a `--fail-fast` CLI flag to terminate the entire grid run on the first runner error.
* **Robust Health Check Initialization**: Ensure a failure to start the health check server (e.g., port in use) is a fatal application error.
* **Configurable Worker Pool**: Add a `--workers` CLI flag to allow users to control the level of concurrency.
* **HCL Meta-Arguments**: Add full support for `count` and `for_each` to enable dynamic looping and conditional execution.
* **`bggo-builder` Tool**: Develop the command-line builder tool to enable a third-party runner ecosystem without forking the main project.

#### Pillar: Insights & Reporting
* **Native OpenTelemetry (OTLP) Export**: Add first-class support for exporting traces and metrics. This is the cornerstone of the "Insights" pillar.
* **Live Terminal UI (TUI)**: Build the rich, interactive terminal dashboard to provide a "cockpit view" of test runs in real-time.
* **Prometheus Metrics Endpoint**: Provide an optional `/metrics` endpoint for easy scraping of performance data.
* **DAG Visualization Command**: Implement a `burstgridgo graph` command to output a visual representation (e.g., Mermaid or DOT format) of a grid's execution plan.

#### Pillar: Expansion & Protocols
* **New Core Runners**: Ship official runners for high-demand protocols, starting with **gRPC** and **WebSockets**.
* **Stateful Session Management**: Introduce a mechanism for sharing state (like auth tokens or session cookies) between runners in a structured way.

---

### Long-Term Vision ✨
* **Distributed Execution**: Architect `burstgridgo` to run in a controller/agent mode to enable massive-scale load tests orchestrated from a single control plane.



Final Architecture Plan
This architecture creates a robust, type-safe contract between the HCL configuration and the Go code. It separates the definition of a runner from its execution, making the system more modular and self-documenting.

1. The Runner Definition (The runner Block)

A runner's public API is now defined in HCL files located within its module directory (e.g., modules/http_request/).

Discovery: The application will find these definitions by recursively scanning the modules/ directory at startup.

Structure: The definition is wrapped in a runner "type" {} block.

Example Definition (modules/http_request/manifest.hcl):

Terraform
runner "http_request" {
  description = "Executes a simple HTTP request."

  input "url" {
    type        = string
    description = "The URL to send the request to."
  }

  input "method" {
    type        = string
    description = "The HTTP method to use."
    optional    = true
    default     = "GET"
  }

  output "status_code" {
    type        = number
    description = "The HTTP status code of the response."
  }

  lifecycle {
    on_run = "OnRunHttpRequest"
  }
}
2. The Execution Instance (The step Block)

A user executes a runner in their grid file by using a step block. This creates an instance of a defined runner.

Structure: A step "type" "name" {} block is used to call a runner.

arguments {} Block: Provides the actual values for the inputs defined in the runner manifest.

Dependencies: Uses depends_on and HCL interpolation (${step.other_step.output.field}) to build the DAG.

Example Execution (my_test.hcl):

Terraform
step "http_request" "get_homepage" {
  arguments {
    url = "https://example.com"
  }
}
3. The Go Handler Implementation

The Go code acts as a library of handler functions that are explicitly registered with the engine.

Registration: Each handler is registered by its string name in a global map using Go's init() function (e.g., engine.Register("OnRunHttpRequest", OnRunHttpRequest)).

Stateful Signatures: The handlers use a standard signature that passes state between lifecycle events.

on_start creates a state object: func(...) (*State, error)

on_run receives that state: func(state *State, ...) (*Output, error)

on_end also receives the state for cleanup: func(state *State) error

Typed I/O: Handlers use native Go structs for inputs and outputs. Output structs must use cty:"snake_case_name" tags to expose their fields to HCL.

4. The Executor's Role (The Bridge)

The executor connects the HCL configuration to the Go logic.

Validation: Before running, the executor performs a pre-flight check, validating the arguments in each step against the input schema of the corresponding runner. This catches errors early.

State Management: It calls the on_start handler, captures the returned *State object, and passes it to on_run and on_end for that specific step instance.

Data Flow:

It decodes the HCL arguments into the typed Go *Input struct for the handler.

After the handler returns a Go *Output struct, the executor uses cty.ToValue() to convert it into a cty.Value. This makes the output available to other steps in the DAG.