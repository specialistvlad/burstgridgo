# burstgridgo

[![Go CI](https://github.com/specialistvlad/burstgridgo/actions/workflows/ci.yml/badge.svg)](https://github.com/specialistvlad/burstgridgo/actions/workflows/ci.yml)
[![codecov](https://codecov.io/github/specialistvlad/burstgridgo/graph/badge.svg?token=SZRP5JPQBC)](https://codecov.io/github/specialistvlad/burstgridgo)
[![Go Report Card](https://goreportcard.com/badge/github.com/specialistvlad/burstgridgo)](https://goreportcard.com/report/github.com/specialistvlad/burstgridgo)
[![Go Reference](https://pkg.go.dev/badge/github.com/specialistvlad/burstgridgo.svg)](https://pkg.go.dev/github.com/specialistvlad/burstgridgo)
[![GitHub release (latest by date)](https://img.shields.io/github/v/release/specialistvlad/burstgridgo)](https://github.com/specialistvlad/burstgridgo/releases)
[![License](https://img.shields.io/github/license/specialistvlad/burstgridgo)](https://github.com/specialistvlad/burstgridgo/blob/main/LICENSE)

<br>

> **Project Status: ⚠️Proof of Concept ⚠️**
>
> This project is under active development. The API and internal architecture are **not yet stable** and are subject to breaking changes. It is not recommended for production use at this stage.

## What is `burstgridgo`?

`burstgridgo` is a powerful, declarative workflow engine that lets you define complex, multi-step processes as code.

It uses a simple, [HCL-based syntax](https://github.com/hashicorp/hcl) (like [Terraform](https://developer.hashicorp.com/terraform)) to define *what* you want to run—such as HTTP requests, gRPC calls, or database queries. `burstgridgo` automatically builds a dynamic dependency graph (DAG) from your definitions to execute everything with maximum concurrency.

It's designed to be the "glue" for complex automation, testing, and data orchestration tasks, such as:

* Simulating real-world user traffic in a load test.
* Orchestrating end-to-end tests for a microservices architecture.
* Automating complex data pipelines or back-office processes.

## Core Features
* **Declarative HCL Workflows:** Define all your resources and execution steps in simple, composable .hcl files.
* **Automatic Concurrency:** Automatically builds a Dependency Graph (DAG) from your HCL to run independent tasks in parallel.
* **Stateful Resource Management:** Define resource blocks (like an http_client) that are created once and shared by multiple steps.
* **Dynamic Fan-out/Fan-in:** Natively supports parallel execution patterns using the count meta-argument and splat operator ([*]) for collecting results.
* **Type-Safe & Validated:** Includes a core type system with support for primitives (string, number), collections (list, map), and objects.
* **Extensible Architecture:** Easily add new capabilities (like http_request or print) through a pluggable "Runner" and "Asset" system.

## Community & Vision
Beyond being a useful tool, this project has two other core goals:

* To Build a Powerful, Open-Source Automation Hub: The long-term vision is to create a flexible, open-source alternative to platforms like Zapier or n8n, with a strong focus on developer-centric tooling (like HCL) and future-looking integrations (like AI/LLM orchestration).
* To Be a Welcoming Place to Collaborate: This project is intentionally built as a place for engineers to learn, experiment, and gain experience in open source. We are actively looking for collaborators who are passionate about Go, graph-based systems, or just building cool developer tools. If you've been wanting to contribute to an open-source project but didn't know where to start, you are welcome here.

## 🚀 Getting Started

This guide will walk you through cloning the repository and running your first workflow from the included examples.

### Prerequisites

* [Go](https://go.dev/doc/install) (version 1.25 or later)
* [Make](https://www.gnu.org/software/make/)
* [Docker](https://docker.io) To build images

### 1. Clone the Repository

First, clone the project to your local machine using `git`:

```sh
git clone https://github.com/specialistvlad/burstgridgo.git
cd burstgridgo
```

### 2. Run an Example

You can run one of the provided examples (located in the /examples folder) using the make run command.

For instance, to run an example from the [examples folder](examples/):
```sh
make run ./examples/http_count_static_fan_in.hcl
```

## Learn More & Contribute
* Roadmap: See our Project Roadmap [Project Roadmap](https://github.com/users/specialistvlad/projects/1/views/2) to learn about planned features and internal design.
* Contributing Guide: Find out how you can help make this project better by reading our [Contributing Guide](CONTRIBUTING.md).

## License
burstgridgo is licensed under the MIT License.
