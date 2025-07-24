# burstgridgo

<div style="text-align: center;">
  <a href="https://github.com/specialistvlad/burstgridgo/actions/workflows/ci.yml">
    <img src="https://github.com/specialistvlad/burstgridgo/actions/workflows/ci.yml/badge.svg" alt="Go CI">
  </a>
  <img src="https://user-images.githubusercontent.com/12345/placeholder.gif" alt="TUI in action">
  <a href="https://github.com/specialistvlad/burstgridgo/graphs/commit-activity">
    <img alt="GitHub commit activity" src="https://img.shields.io/github/commit-activity/m/specialistvlad/burstgridgo">
  </a>
  <a href="https://github.com/specialistvlad/burstgridgo/issues">
    <img alt="GitHub open issues" src="https://img.shields.io/github/issues/specialistvlad/burstgridgo">
  </a>
  <a href="https://github.com/specialistvlad/burstgridgo/pulls">
    <img alt="GitHub open pull requests" src="https://img.shields.io/github/issues-pr/specialistvlad/burstgridgo">
  </a>
  <a href="https://github.com/specialistvlad/burstgridgo/blob/main/LICENSE">
    <img alt="License" src="https://img.shields.io/github/license/specialistvlad/burstgridgo">
  </a>
</div>

 <br>

**⚠️ Important Note: Project Status ⚠️**

The project is currently under active development. The API and internal architecture are **not yet stable** and are subject to breaking changes. This project is not recommended for external production use; it is intended for testing and development of `burstgridgo` itself.


`burstgridgo` is a Go-native, declarative load testing tool for simulating real-world, protocol-aware workflows using HCL.


## Core Pillars
* **Declarative Grids (HCL)**: Define multi-protocol workflows (e.g., REST API -> Socket.IO -> S3) in simple, composable HCL files. Your test plans become readable, versionable infrastructure.
* **Intelligent Concurrency (DAG)**: The tool automatically builds a dependency graph (DAG) from your grid, running independent modules in parallel for maximum efficiency. Dependencies are inferred automatically from variable usage.
* **First-Class Insights (O11y + TUI)**: Testing is about understanding, not just running. Get a "cockpit view" with a live-updating Terminal UI and export rich telemetry via native OpenTelemetry to your existing observability stack (Grafana, Datadog, etc.).
* **Extensible by Design (Go Runners)**: Missing a protocol? `burstgridgo` is built on a simple Go `Runner` interface. Implement your own logic in a `.go` file, and you can immediately use it in your HCL.

## Dynamic Workflows
Create dynamic and realistic scenarios using HCL meta-arguments.
* **Looping**: Use `count` or `for_each` to run a task multiple times in parallel, perfect for generating load or testing a list of endpoints.
* **Branching**: Use a ternary expression with `count` (`count = condition ? 1 : 0`) to conditionally include a module in your workflow.

## Getting Started
Prerequisites: **Docker** and **Make**.

To run a grid for development with live-reloading, use the following command:
```sh
# This example runs the http_request.hcl grid
make dev grid=examples/http_request.hcl
```

## Example Workflow
The following grid defines a workflow with multiple dependent HTTP requests.
```hcl
# File: examples/http_request.hcl

step "http_request" "first" {
  arguments {
    url = "https://httpbin.org/get"
  }
}

step "http_request" "second" {
  arguments {
    url = "https://httpbin.org/delay/1"
  }
  depends_on = ["first"]
}

step "http_request" "third" {
  arguments {
    url = "https://httpbin.org/delay/2"
  }
  depends_on = ["first"]
}

step "http_request" "final" {
  arguments {
    url    = "https://httpbin.org/post"
    method = "POST"
  }
  depends_on = [
    "second",
    "third",
  ]
}
```

This configuration generates the following execution graph:
```mermaid
graph TD
    Start((Start)) --> first_request;
    first_request --> second_request;
    first_request --> third_request;
    second_request --> final_request;
    third_request --> final_request;
    final_request --> End((End));
```

## Learn More
* **Roadmap & Vision**: [See where the project is headed.](./ROADMAP.md)
* **Architecture Deep Dive**: [Learn how `burstgridgo` works internally.](./docs/ARCHITECTURE.md)
* **Contributing Guide**: [Find out how you can help.](./CONTRIBUTING.md)
