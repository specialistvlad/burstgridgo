burstgridgo
[BADGE: CI_Status] [BADGE: Code_Coverage] [BADGE: Go_Report_Card] [BADGE: License]

Define complex load tests as code. Get insights, not just output.

[VISUAL: TUI_in_action.gif]

burstgridgo is a Go-native, declarative load testing tool for simulating real-world, protocol-aware workflows using HCL.

Core Pillars
Declarative Grids (HCL)

Define multi-protocol workflows (e.g., REST API -> Socket.IO -> S3) in simple, composable HCL files. Your test plans become readable, versionable infrastructure.

Intelligent Concurrency (DAG)

The tool automatically builds a dependency graph (DAG) from your grid, running independent modules in parallel for maximum efficiency. Dependencies are inferred automatically from variable usage.

First-Class Insights (O11y + TUI)

Testing is about understanding, not just running. Get a "cockpit view" with a live-updating Terminal UI and export rich telemetry via native OpenTelemetry to your existing observability stack (Grafana, Datadog, etc.).

Extensible by Design (Go Runners)

Missing a protocol? burstgridgo is built on a simple Go Runner interface. Implement your own logic in a .go file, and you can immediately use it in your HCL.

Dynamic Workflows
Create dynamic and realistic scenarios using HCL meta-arguments.

Looping: Use count or for_each to run a task multiple times in parallel, perfect for generating load or testing a list of endpoints.

Branching: Use a ternary expression with count (count = condition ? 1 : 0) to conditionally include a module in your workflow.

Getting Started
Prerequisites: Docker and Make.

To run a grid for development with live-reloading, use the following command:

[CODE_SNIPPET: make_dev_command]

Example Workflow
The following grid defines a simple user authentication flow. First, it logs in to get a token, and then it uses that token to fetch a user profile. A separate health check runs in parallel.

[CODE_SNIPPET: hcl_workflow_example]

This configuration generates the following execution graph:

[CODE_SNIPPET: mermaid_dag_diagram]

Learn More
Roadmap & Vision: See where the project is headed.

Architecture Deep Dive: Learn how burstgridgo works internally.

Contributing Guide: Find out how you can help.

