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

* **Evolve the HCL Syntax from `module` to `runner`**:
    * Deprecate the generic `module` block in favor of a more explicit `runner "my_runner_instance" {}` block. This clarifies the configuration's intent.
    * Introduce a `lifecycle {}` block with `on_start`, `on_run`, and `on_end` attributes to map to specific Go functions.
    * Add `input {}` and `output {}` schema blocks within the `runner` definition to create a typed, self-documenting contract between HCL and Go.
    * Standardize on an `arguments {}` block for passing data, which will be decoded into the handler's input struct.

* **Implement New Go Handler Signature & Lifecycle**:
    * Redefine the primary `engine.Runner` interface. The current `Run(m Module, ctx *hcl.EvalContext) (cty.Value, error)` will be replaced.
    * The new signature will be: `func OnRun(ctx context.Context, runnerCtx *RunnerContext, input *RunnerInput) (*RunnerOutput, error)`.
        * **`context.Context`**: For cancellation, timeouts, and deadlines. This will be plumbed from `main` through the executor to every handler, making the system robust and responsive.
        * **`*RunnerContext`**: A new struct containing execution metadata (e.g., runner ID, status, logger instance).
        * **`*RunnerInput` / `*RunnerOutput`**: Auto-generated, type-safe structs based on the HCL `input` and `output` schemas, eliminating manual `cty.Value` conversion.

* **Overhaul the Executor to Support the New Architecture**:
    * Modify the `dag.Executor` to create and manage the `context.Context` for each run.
    * The executor will be responsible for decoding the HCL `arguments` block into the specific `*RunnerInput` struct for the handler and processing the returned `*RunnerOutput`.

#### 2. Fix Critical Build & CI/CD Flaws
The current CI and build process has several critical issues that undermine reliability.

* **Optimize `Makefile` for Development**:
    * **Problem**: The `make dev` target rebuilds the dev Docker image on every invocation, which is slow and unnecessary.
    * **Fix**: Refactor the `dev` target to perform a one-time build of the dev image (if it doesn't exist) and then simply `docker run` on subsequent calls, mounting the code as a volume.

#### 3. Address Performance & Concurrency
Improve the performance and scalability of the core engine and runners.

* **Eliminate Executor Concurrency Bottleneck**:
    * **Problem**: The `dag.Executor` uses a single `nodeMutex` to manage state changes and check for ready dependents. This creates a contention point on graphs with high fan-out.
    * **Fix**: Replace the global mutex with a lock-free approach. Each node will have an atomic integer acting as a dependency counter. When a node finishes, it will atomically decrement the counter of its dependents. A dependent is scheduled to run when its counter reaches zero.

* **Implement Shared HTTP Clients**:
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