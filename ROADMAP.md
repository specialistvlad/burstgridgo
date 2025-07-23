🧭 Project Roadmap
This document outlines the development roadmap for burstgridgo. Our vision is to create the best tool for defining complex load tests as code and turning the results into actionable insights.

Foundation: Engine & Developer Experience (DX)
This pillar focuses on making the core tool robust, stable, and easy for developers to use and extend.

Implement HCL Meta-Arguments: Add full support for count and for_each to enable dynamic looping and conditional execution of modules.

Graceful Cancellation & Timeouts: Integrate context.Context throughout the engine for robust cancellation and module-level timeouts.

Create bggo-builder Tool: Develop the command-line builder tool to enable a third-party runner ecosystem, allowing users to compile custom binaries with their own runners.

Build Doc-Gen Tool: Create the Go program that automatically generates runner documentation from source code comments and examples.

Configurable Worker Pool: Add a --workers flag to allow users to control the level of test concurrency.

Insights: Observability & Reporting
This pillar focuses on our "insights, not just output" philosophy, ensuring that test results are rich, interactive, and easy to integrate.

Live Terminal UI (TUI): Build the rich, interactive terminal dashboard to provide a "cockpit view" of test runs with live metrics, DAG status, and log streaming.

Native OpenTelemetry (OTLP) Export: Add first-class support for exporting traces, metrics, and logs via OTLP for seamless integration with platforms like Grafana, Datadog, and Honeycomb.

Prometheus Metrics Endpoint: Provide an optional /metrics endpoint for easy scraping by any Prometheus instance.

DAG Visualization Command: Implement a burstgridgo graph command to output a visual representation of a grid's execution plan.

Expansion: Protocol & Workflow Support
This pillar focuses on broadening the tool's capabilities by adding support for more protocols and complex workflow patterns.

New Core Runners: Ship official runners for high-demand protocols, starting with gRPC and WebSockets, followed by message queue systems like Kafka and NATS.

Stateful Session Management: Introduce a mechanism for sharing state (like auth tokens or cookies) across modules to simplify the simulation of realistic user sessions.

Scripting "Escape Hatch": Add a script runner (using a library like goja) to allow for complex logic or data manipulation using JavaScript.

Long-Term Vision
Distributed Execution: Architect burstgridgo to run in a controller/agent mode to enable massive-scale load tests that are not limited by a single machine's resources.