# burstgridgo

<div style="text-align: center;">
  <a href="https://github.com/specialistvlad/burstgridgo/actions/workflows/ci.yml">
    <img src="https://github.com/specialistvlad/burstgridgo/actions/workflows/ci.yml/badge.svg" alt="Go CI">
  </a>
  <a href="https://codecov.io/github/specialistvlad/burstgridgo" >
    <img src="https://codecov.io/github/specialistvlad/burstgridgo/graph/badge.svg?token=SZRP5JPQBC"/>
  </a>
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

> **⚠️ Project Status: Proof of Concept ⚠️**
>
> This project is under active development. The API and internal architecture are **not yet stable** and are subject to breaking changes. It is not recommended for production use at this stage.

Experimental tool designed for simulating real-world workflows using HCL.

What does it mean?

## Core Features
* **Declarative Workflows**: Define complex, multi-protocol test scenarios (grids) in simple, composable HCL files.
* **Unified Configuration**: All `.hcl` files are loaded recursively and treated as a single collection. No explicit imports are needed between your local files.
* **Concurrency in mind**: Automatically builds a dependency graph (DAG) from your workflows, running independent tasks in parallel while correctly resolving dependencies.


### Example Grid
The following grid defines a workflow with multiple dependent HTTP requests.
```hcl
# File: examples/http_count_static_fan_in.hcl
# This example demonstrates a static fan-out/fan-in pattern.
# It runs a static count (10) of HTTP requests in parallel and then
# uses the splat operator [*] to collect all results into a final step.
# To run this example use a command like: `make run ./examples/http_count_static_fan_in.hcl`

# 1. Define a stateful, shared resource.
resource "http_client" "shared" {
  arguments {
    timeout = "45s"
  }
}

# 2. Define steps that consume the shared resource.
step "http_request" "first" {
  uses {
    client = resource.http_client.shared
  }
  arguments {
    url = "https://httpbin.org/get"
  }
}

# 3. These two steps are now replaced by a single block.
step "http_request" "delay_requests" {
  count = 10

  uses {
    client = resource.http_client.shared
  }
  arguments {
    url = "https://httpbin.org/delay/${count.index + 1}"
  }
  depends_on = ["http_request.first"]
}


# 4. This "fan-in" step collects and prints the output from ALL instances.
# It demonstrates the splat operator working on the dynamic group.
step "print" "show_all_results" {
  arguments {
    # This implicitly depends on the entire "delay_requests" group finishing.
    # The splat operator collects the 'output' from every instance into a list.
    input = step.http_request.delay_requests[*].output
  }
}
```

## Getting Started

### Production (Coming Soon)
`docker run ...`
*(Instructions will be added upon the first stable release.)*

### Development
Prerequisites: **Docker** and **Make**.

To run a test grid with live-reloading, clone the repository and execute the following command from the root directory:

`make dev grid=examples/http_request.hcl`

This command mounts the current directory into the container, allowing you to edit files and see changes instantly.

---

## Features

*(✅ Implemented | 🚧 In Progress | 💡 Planned)*

* **✅ Core Foundation for POC**:
  * ✅ CLI Interface
  * ✅ HCL Configuration Loading
  * ✅ DAG Graph Building & Execution
  * ✅ Concurrent Execution Engine
  * ✅ Implicit & Explicit Dependencies (Fan-in / Fan-out)
  * ✅ Basic Module & Runner Support
  * ✅ Docker Image for Distribution
* **✅ Type System & Validation**:
  * ✅ **Primitives:** `string`, `number`, `bool`
  * ✅ **Collections:** `list(T)`, `map(T)`, `set(T)`
  * ✅ **Objects:** Structurally-typed `object({key=type, ...})` and generic `object({})`
* **✅ Pluggable & Unified Configuration**:
  * ✅ Extensible Runner/Asset architecture for stateless and stateful operations. (See `ADR-001`)
  * ✅ Format-agnostic configuration system treats all `.hcl` files as a single collection. (See `ADR-007`)
* **✅ Stateful Resource Management**:
  * ✅ Full lifecycle for `resource` blocks, including creation, destruction, and instance sharing via the `uses` block.
* **✅ Execution Engine**:
  * ✅ Fail-Fast Execution correctly cancels all running tasks as soon as one node fails.
* **✅ Development & CI/CD**:
  * ✅ Containerized development environment with live-reloading.
  * ✅ Core internal packages refactored for maintainability (`app`, `cli`, `config`, `hcl`, `dag`, `executor`). (See `ADR-002`)
  * ✅ Comprehensive integration test suite validating core features and concurrency patterns. (See `ADR-003`)
* **Website**:
  * Landing page
  * Documentation
  * Auto publishing documentation
* **💡 Dynamic Workflows & Meta-Arguments**:
  * ✅ Static `count` parameter (resolution of the DAG at build time phase)
  * 🚧 Dynamic `count` parameter (resolution of the DAG in runtime)
  * 🚧 Static `for_each` parameter (resolution of the DAG at build time phase)
  * 🚧 Dynamic `for_each` parameter (resolution of the DAG in runtime)
  * 💡 Advanced dependency patterns for collections: All-to-One, One-to-One, Specific-to-One, and Any-to-One (Race).
* **💡 Execution Controls**:
  * 💡 **Conditional Execution**: `if` meta-argument to conditionally skip steps.
  * 💡 **Concurrency Limiting**: `concurrency {}` block to control parallelism within loops.
  * 💡 **Delays & Timeouts**: `delay_before`, `delay_after`, and `timeouts {}` blocks.
  * 💡 **Automatic Retries**: `retry {}` block to re-run failed steps with configurable attempts and backoff.
  * 💡 **Execution cache**: If stateless(no side effects) step has input parameters same as one in the cache before - it will not be executed, but instead output will be taken from the cache.
* **Storage backends**
  * 💡 **In memory**
  * 💡 **Redis**
* **Distributed running**
  * 💡 **Multi instance**
* **💡 Configuration & Usability**:
  * 💡 **Definition Scoping**: A `scope` meta-argument (`local`, `module`, `workspace`, `global`) to control visibility and prevent name collisions.
  * 💡 **Sensitive Data Handling**: A `sensitive = true` flag to redact secret values from all logs.
  * 💡 **versioning system**: inside hcl
* **💡 Insights & Reporting**:
  * 💡 **Native OpenTelemetry (OTLP) Export**: First-class support for exporting traces and metrics.
  * 💡 **Live Terminal UI (TUI)**: An interactive terminal dashboard for real-time test monitoring. (See `ADR-004`)
  * 💡 **DAG Visualization**: A `bggo graph` command to output a visual graph (Mermaid or DOT format).
  * 💡 **Prometheus Metrics**: An optional `/metrics` endpoint for scraping performance data.
* **🚧 Logging**: Structured Logger Implementation.
* **💡 Module System**:
  * 💡 **External Module System**: Revisit the module system to allow for dynamic, third-party module registration.
  * 💡 **Remote communication interface**: To be able to isolate modules from the code.
  * 💡 **Release System**: Streamlined process for versioning and releasing the application.
* **HCL features**
  * ✅ **Expression Support**
  * 💡 **Splat operator support**
  * 💡 **Variables support**
  * 💡 **Global Variables**: Pass variables via CLI flags (`-var 'key=value'`, `-var-file="vars.hcl"`), `-var-file="vars.json"`).

## Modules
* **Utilities**:
  * ✅ `env_vars`
  * ✅ `print`
  * 💡 `ls dir`
  * 💡 `execute script`
  * 💡 `cmd`: A runner to execute local shell commands, capturing stdout, stderr, and the exit code.
* **HTTP**:
  * ✅ Basic `http_client` asset and `http_request` runner.
  * 💡 Add support for custom `headers`, request `body`, `query_params`, and `form_data`.
  * 💡 Introduce helpers for common authentication schemes (e.g., Bearer Token, Basic Auth).
* **Socket.IO**:
  * ✅ A native client for Socket.IO interactions.
* **S3**:
  * 💡 Basic file upload runner.
  * 💡 Expand to support standard S3 API actions (`put_object`, `get_object`) using credentials.
* **gRPC**:
  * 💡 A dedicated runner for making unary and streaming gRPC calls.
* **WebSockets**:
  * 💡 A native runner and asset for interacting with standard WebSocket services.
* **Databases & Caches**:
  * 💡 `redis`: A runner and asset for interacting with a Redis server.
  * 💡 `postgres`: A runner and asset for executing queries against a PostgreSQL database.
  * 💡 `mongo`: A runner and asset for executing commands against a MongoDB database.
* **Message Queues**:
  * 💡 `rabbitmq`: A runner and asset for publishing and consuming messages from RabbitMQ.
  * 💡 `kafka`: A runner and asset for producing and consuming messages from Kafka topics.
* **Integrations & Servers**:
  * 💡 `slack`: A runner for sending notifications to a Slack webhook.
  * 💡 General WebHook server to control grid execution.
  * 💡 `MCP Server`

## Learn More & Contribute

* **Architecture Deep Dive**: [Learn how `burstgridgo` works internally.](./internal/Readme.md)
* **Contributing Guide**: [Find out how you can help make this project better.](./CONTRIBUTING.md)
