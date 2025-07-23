# 🧭 Project Roadmap
This document outlines the development roadmap for `burstgridgo`. Our vision is to create the best tool for defining complex load tests as code and turning the results into actionable insights.

This roadmap is a living document. Priorities may shift based on community feedback and technical needs.

---

### Completed ✅
* **Core Engine v1**: Foundational DAG-based executor with HCL parsing.
* **Extensible Runner Architecture**: The core `engine.Runner` interface and dynamic registration system.
* **Initial Core Runners**: Shipped runners for `http-request`, `socketio`, `s3`, `print`, and `env_vars`.
* **Containerized DX**: Multi-stage `Dockerfile` and `Makefile` for reproducible development and production builds.

---

### Next Up: Addressing Critical Issues 🎯
These are the highest-priority items for the next release cycle. They focus on **addressing foundational architectural flaws** and improving the project's stability and correctness.

* **Robust Execution with `context.Context`**: Integrate `context.Context` throughout the engine, from `main` through the executor and into every runner's `Run` method. This is the top priority for enabling graceful cancellation, timeouts, and overall stability.
* **Fix CI/CD & Build Workflow**: Correct critical bugs in the build and CI process. This includes removing `*_test.go` from `.dockerignore` so tests actually run in the build, aligning the `go-version` in CI with `go.mod`, and optimizing the `Makefile` `dev` target to avoid rebuilding the image on every run.
* **Fix Executor Concurrency Bottleneck**: Remove the global mutex (`nodeMutex`) in the DAG executor and replace it with a lock-free approach using atomic counters to prevent performance bottlenecks on graphs with high fan-out.
* **Native OpenTelemetry (OTLP) Export**: Add first-class support for exporting traces and metrics. This is the cornerstone of the "Insights" pillar.
* **Extensible Runner Architecture v0.2**: Enhance the `engine.Runner` interface with optional `Setup`, `Teardown`, and `Validate` methods (all accepting `context.Context`) to allow developers more control over runner lifecycle, resource management, and configuration validation.

---

### Future Ideas & Backlog 💡
This is a list of features and less critical issues that are planned but not yet scheduled. They are grouped by strategic pillar.

#### Pillar: Foundation & Developer Experience (DX)
* **Comprehensive Test Coverage**: Implement a robust, table-driven test suite for the `dag` package, covering implicit/explicit dependencies, error conditions, and complex graphs. Add full unit test coverage for complex runner logic.
* **Efficient Core Runners**: Refactor runners (`http-request`, `s3`) to use a shared, package-level `http.Client` to improve performance through connection reuse.
* **Standardized Error Handling & Fail-Fast Mode**: Make error handling consistent across all packages. Introduce a `--fail-fast` flag to terminate the entire grid run on the first module error.
* **Robust Health Check Initialization**: Ensure that a failure to start the health check server (e.g., port in use) is a fatal error for the application, preventing it from running silently in a non-monitorable state.
* **Configurable Worker Pool**: Add a `--workers` flag to allow users to control the level of test concurrency.
* **HCL Meta-Arguments**: Add full support for `count` and `for_each` to enable dynamic looping and conditional execution of modules.
* **Explicit Module Inputs/Outputs (Terraform-like)**: Introduce structured HCL blocks (e.g., `input { ... }` and `output { ... }`) within module definitions or similar, allowing runners to explicitly define their required inputs and the schema of their produced outputs, similar to Terraform. This will enhance configuration readability, enable early validation, and standardize data flow.
* **`bggo-builder` Tool**: Develop the command-line builder tool to enable a third-party runner ecosystem.
* **`doc-gen` Tool**: Create the Go program that automatically generates runner documentation from source code comments.

#### Pillar: Insights & Reporting
* **Live Terminal UI (TUI)**: Build the rich, interactive terminal dashboard to provide a "cockpit view" of test runs.
* **Prometheus Metrics Endpoint**: Provide an optional `/metrics` endpoint for easy scraping.
* **DAG Visualization Command**: Implement a `burstgridgo graph` command to output a visual representation (e.g., Mermaid or DOT format) of a grid's execution plan.

#### Pillar: Expansion & Protocols
* **New Core Runners**: Ship official runners for high-demand protocols, starting with **gRPC** and **WebSockets**.
* **Stateful Session Management**: Introduce a mechanism for sharing state (like auth tokens or cookies) across modules.
* **Scripting "Escape Hatch"**: Add a `script` runner (using a library like `goja`) to allow for complex logic using JavaScript.

---

### Long-Term Vision ✨
* **Distributed Execution**: Architect `burstgridgo` to run in a controller/agent mode to enable massive-scale load tests.