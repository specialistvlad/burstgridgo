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

**⚠️ Important Note: Project Status ⚠️**

This project is currently under active development. The API and internal architecture are **not yet stable** and are subject to breaking changes. This project is not recommended for external production use; it is a POC (proof of concept).

**⚠️ Important Note: Project Status ⚠️**



`burstgridgo` is a Go-native, declarative load testing tool for simulating real-world, protocol-aware workflows(grids) using HCL.

## Core Features
* **✅ Declarative Programming**: Define complex, multi-protocol grids in simple, composable HCL files. 
* **✅ Unified Configuration**: The loader treats all `.hcl` files from all paths recursively as a single collection. It imports all your definitions. No need imports. 
* **✅ Ideological Concurrent and .....**: The builds a dependency graph (DAG) from all grids, running independent tasks in parallel, controlling the flow of execution and deps resolution. Dependencies are inferred automatically from variable usage.
* **✅ Extensible by Design**: Missing a runner? It is a simple to add your own one. Welcome to contribute. is built on a simple Go `Module` interface. Implement your own logic and immediately use it as a module in your grid.
* **✅ Minimal setup to start**: There is only one depency to start project quickly - docker. If you want more comfortable environment then `make` help you, or for fully local development you need `go` toolkit. Any of them works anyways.

## Learn More
* **Architecture Deep Dive**: [Learn how it works internally.](./internal/Readme.md)

## Example Grid
The following grid defines a workflow with multiple dependent HTTP requests.
```hcl
# File: examples/http_concurrent_requests.hcl

step "http_request" "httpbin" {
  count = 10

  concurrency {
    limit = 3
  }

  retry {
    attempts = 2
    delay    = 2s
  }

  arguments {
    url = "https://httpbin.org/get?$request={count.index}"
  }
}

step "print" "wait_each" {
  arguments {
    input = "Request=${count.index} code=${http_request.httpbin[each].output.status_code}"
  }
}

step "print" "wait_all" {
  arguments {
    input = "We made ${count(http_request.httpbin.output)} requests!"
  }
}
```


## How to run
Prerequisites: **Docker** and **Make**.


### Production (coming soon)

#### docker
```sh
docker run .... 
```

#### compose
```yaml

```

#### other method
```
What can be here?
```

### Development
To run a grid for development with live-reloading in docker.

Note: You need to pull the repository first and run it from repository. Because it injects the code from current folder.
, use the following command:
```sh
# Start local dev environment 
make dev grid=examples/http_request.hcl
```

### 🧭 Project Roadmap

1. ✅ (Jul 27 2025) - Project start
2. 🚧 (In progress) - POC Preview v0.1-dev
2. 💡 (Planned) - POC Preview v0.2-dev
2. 💡 (Planned) - POC Preview v0.3-dev

---

### Features
*(✅ Implemented | 🚧 In Progress | 💡 Planned)*

* **✅ Foundation for POC**:
  * ✅ CLI
  * ✅ HCL loading
  * ✅ DAG Graph building
  * ✅ Concurent Execution
  * ✅ Dependencies
    * ✅ Fan in
    * ✅ Fan out
    * ✅ Inmplicit/Explicit
  * ✅ HCL Expressions
  * ✅ Docker image
  * ✅ Basic modules support
  * ✅ Type System & Validation
    * ✅ **Primitives:** `string`, `number`, `bool`
        * ✅ **Collections:** `list(T)`, `map(T)`, `set(T)` where `T` is one of the primitive types.
        * ✅ **Objects:**
            * ✅ **`object({key=type, ...})`**: A structurally-typed object that maps to a Go `struct`.
            * ✅ **`object({})`**: A generic object that maps to a Go `map[string]any`.
  * Looger
* **✅ Extensible Runner/Asset Architecture**: The system for defining stateless `runners` and stateful `assets` via HCL manifests and registering their Go implementations is complete. (See `ADR-001`)
* **✅ Stateful Resource Management**: The full lifecycle for `resource` blocks—including creation, destruction, and sharing instances between steps via the `uses` block—is implemented.
* **✅ Pluggable & Unified Configuration**: The configuration loading system has been refactored to be format-agnostic. It now treats all `.hcl` files as a single, unified collection, allowing definitions and instances to be co-located. (See `ADR-007`)
* **✅ Fail-Fast Execution**: The executor correctly cancels all running tasks as soon as one node fails, ensuring rapid feedback on errors.
* **✅ Containerized Development Environment**: A multi-stage `Dockerfile` and `Makefile` provide a one-command setup for a live-reloading development environment (`make dev`).
* **✅ Core Internal Refactoring**: The application has been successfully refactored into decoupled internal packages (`app`, `cli`, `config`, `hcl`, `dag`, `executor`) for improved maintainability. (See `ADR-002`)
* **✅ Comprehensive Integration Test Suite**: A robust integration test suite is in place, validating core HCL features, concurrency patterns, and error handling. (See `ADR-003`)
* Release system
* **💡 External Module System**: Revisit the module system to allow for dynamic, third-party module registration.
* **💡 Dynamic Workflows & Meta-Arguments**: Full support for HCL features like `count`, `for_each`, and and additional logic (meta-arguments) to create multiple instances of steps and resources from a single configuration block. This includes support for four key dependency patterns:
    * **All-to-One**: A step can wait for all instances in a collection to complete (e.g., by referencing the collection's `output`).
    * **One-to-One (Templating)**: A step can be implicitly cloned to run for each instance in a collection (e.g., using an `[each]` reference).
    * **Specific-to-One**: A step can wait for a single, specific instance from a collection (e.g., via a direct index like `[3]`).
    * **Any-to-One (Race)**: A step can wait for just the first instance in a collection to complete (e.g., via a special `first_output` attribute).
* Declarative validation
  * BLOBS
  * REGEX
* **Conditional Execution**: Add an `if` meta-argument to conditionally skip the execution of a step or resource based on a boolean expression.
* **Concurrency Limiting**: Implement a `concurrency {}` block to limit the number of simultaneous executions within a `count` or `for_each` loop.
* **Execution Delays**: Introduce `delay_before` and `delay_after` meta-arguments to pause execution for a specified duration before or after a step runs.
* **Execution Timeouts**: Add a `timeouts {}` block to enforce a maximum execution time for any given step or resource.
* **Automatic Retries**: Introduce a `retry {}` block to automatically re-run a failed step with a configurable number of attempts and delay.
* **Global Variables**: Allow passing global variables into a run via CLI flags, such as `-var 'key=value'` and `-var-file="path/to/vars.hcl"`.
* **Definition Scoping**: Introduce a `scope` meta-argument (`local`, `module`, `global`) on `runner` and `asset` definitions to control their visibility and prevent naming collisions across files and folders.
* **Sensitive Data Handling(PII)**: Add a `sensitive = true` flag to input and output definitions to ensure secret values (like passwords or API keys) are redacted from all logs.
* **Insights & Reporting**
  * **💡 Native OpenTelemetry (OTLP) Export**: Add first-class support for exporting traces and metrics to OTLP-compatible backends like Jaeger or Honeycomb.
  * **💡 Live Terminal UI (TUI)**: Build an interactive terminal dashboard for a real-time view of test execution, including throughput, latency, and errors. (See `ADR-004`)
  * **💡 DAG Visualization Command**: Implement a `bggo graph` command to output a visual representation of the execution graph (e.g., in Mermaid or DOT format).
  * **💡 Prometheus Metrics Endpoint**: Provide an optional `/metrics` endpoint for scraping performance data during a test run.


* **Modules**
  * **Utilities**:
    * ✅ `env_vars`
    * ✅ `print`
  * **✅ HTTP**: http client
    * **💡 `http_request` Runner**:
    * ✅ Basic http client
    * Add support for custom `headers`, request `body`, `query_params`, and `form_data`.
    * Introduce helpers for common authentication schemes (e.g., Bearer Token, Basic Auth).
  * **✅ Socket.IO**: Socket.io client
  * **💡 S3**: s3 file upload
    * **💡 `s3` basic file upload**:
    * Expand beyond pre-signed URLs to support standard S3 API actions (`put_object`, `get_object`, `delete_object`, `list_objects`) using credentials.
    * Refactor to use the shared `http_client` asset for connection reuse.
  * General WebHook server to control grids execution (from Slack)
  * ls dir
  * MCP Server
  * Execute script
  * **💡 gRPC**: A dedicated runner for making unary and streaming gRPC calls.
  * **💡 WebSockets**: A native runner and asset for interacting with standard WebSocket services (distinct from Socket.IO).
  * **💡 `redis`**: A runner and asset for interacting with a Redis server (GET, SET, PUBLISH, etc.).
  * **💡 `postgres`**: A runner and asset for executing queries against a PostgreSQL database.
  * **💡 `mongo`**: A runner and asset for executing commands against a MongoDB database.
  * **💡 `cmd`**: A runner to execute local shell commands, capturing stdout, stderr, and the exit code.
  * **💡 `slack`**: A runner for sending notifications to a Slack webhook.
  * **💡 `rabbitmq`**: A runner and asset for publishing and consuming messages from RabbitMQ.
  * **💡 `kafka`**: A runner and asset for producing and consuming messages from Kafka topics.

## Learn More
* **Architecture Deep Dive**: [Learn how `burstgridgo` works internally.](./internal/Readme.md)
* **Contributing Guide**: [Find out how you can help.](./CONTRIBUTING.md)