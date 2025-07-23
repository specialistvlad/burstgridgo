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

### Next Up 🎯
These are the highest-priority items for the next release cycle. They focus on robustness and core observability.

* **Graceful Cancellation & Timeouts**: Integrate `context.Context` throughout the engine for robust cancellation and to enable module-level timeouts. This is critical for stability.
* **Native OpenTelemetry (OTLP) Export**: Add first-class support for exporting traces and metrics. This is the cornerstone of the "Insights" pillar.
* **Configurable Worker Pool**: Add a `--workers` flag to allow users to control the level of test concurrency for performance tuning.
* **DAG Visualization Command**: Implement a `burstgridgo graph` command to output a visual representation (e.g., Mermaid or DOT format) of a grid's execution plan.

---

### Future Ideas & Backlog 💡
This is a list of features that are planned but not yet scheduled for a specific release. They are grouped by their strategic pillar.

#### Pillar: Foundation & Developer Experience (DX)
* **HCL Meta-Arguments**: Add full support for `count` and `for_each` to enable dynamic looping and conditional execution of modules.
* **`bggo-builder` Tool**: Develop the command-line builder tool to enable a third-party runner ecosystem, allowing users to compile custom binaries without cloning the main project.
* **`doc-gen` Tool**: Create the Go program that automatically generates runner documentation from source code comments and examples.

#### Pillar: Insights & Reporting
* **Live Terminal UI (TUI)**: Build the rich, interactive terminal dashboard to provide a "cockpit view" of test runs with live metrics, DAG status, and log streaming.
* **Prometheus Metrics Endpoint**: Provide an optional `/metrics` endpoint for easy scraping by any Prometheus instance.

#### Pillar: Expansion & Protocols
* **New Core Runners**: Ship official runners for high-demand protocols, starting with **gRPC** and **WebSockets**, followed by message queue systems like **Kafka** and **NATS**.
* **Stateful Session Management**: Introduce a mechanism for sharing state (like auth tokens or cookies) across modules to simplify the simulation of realistic user sessions.
* **Scripting "Escape Hatch"**: Add a `script` runner (using a library like `goja`) to allow for complex logic or data manipulation using JavaScript for maximum flexibility.

---

### Long-Term Vision ✨
* **Distributed Execution**: Architect `burstgridgo` to run in a controller/agent mode to enable massive-scale load tests that are not limited by a single machine's resources.