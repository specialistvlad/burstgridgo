# burstgridgo

[![Go CI](https://github.com/specialistvlad/burstgridgo/actions/workflows/ci.yml/badge.svg)](https://github.com/specialistvlad/burstgridgo/actions/workflows/ci.yml)
[![codecov](https://codecov.io/github/specialistvlad/burstgridgo/graph/badge.svg?token=SZRP5JPQBC)](https://codecov.io/github/specialistvlad/burstgridgo)
[![Go Report Card](https://goreportcard.com/badge/github.com/specialistvlad/burstgridgo)](https://goreportcard.com/report/github.com/specialistvlad/burstgridgo)
[![Go Reference](https://pkg.go.dev/badge/github.com/specialistvlad/burstgridgo.svg)](https://pkg.go.dev/github.com/specialistvlad/burstgridgo)
[![GitHub release (latest by date)](https://img.shields.io/github/v/release/specialistvlad/burstgridgo)](https://github.com/specialistvlad/burstgridgo/releases)
[![License](https://img.shields.io/github/license/specialistvlad/burstgridgo)](https://github.com/specialistvlad/burstgridgo/blob/main/LICENSE)

<br>

# DISCLAIMER
**Project Status: ⚠️ MVP Stage 1 ⚠️**
>
> Thanks for checking out BurstGridGo!
> I’m currently the sole maintainer of this project, but I welcome > any form of contribution — from questions to code.
> This project is a great entry point if you’re new to open source.
> This project is under active development. The API and internal architecture are **not yet stable** and are subject to breaking changes. It is not recommended for any real use at this stage.

## What is `burstgridgo`?

BurstGridGo is an open-source experiment on its way to becoming a **production-grade automation and orchestration engine** — a declarative system built in Go for developers who care about clarity, scalability, and modern engineering principles.  
It blends **declarative design**, **visual control**, and **AI-native automation** to help engineers define and orchestrate complex workflows with precision and insight.  
The project’s long-term mission is to turn today’s experimentation into tomorrow’s reliable platform.

## ⚙️ Refined Strategic Pillars

1. **Declarative Core**  
   Define *what* to run, not *how*. The engine builds and executes the DAG automatically from clear, type-safe HCL.

2. **Modular Ecosystem**  
   A free, open marketplace for reusable modules — from HTTP and gRPC to AI/LLM runners — designed for easy discovery and contribution.

3. **Human-Grade Experience**  
   Power users get a rich **TUI**; everyone else gets a **Figma-quality Web UI** with drag-and-drop workflow design, live logs, and visual scaling.

4. **Scalable Architecture**  
   Built to handle both local runs and distributed workloads, scaling seamlessly across cores or clusters for load testing or orchestration.

5. **AI-Native Automation**  
   BurstGridGo will bridge traditional automation with intelligent systems — integrating directly with modern AI protocols (MCP, OpenAI, Anthropic, etc.) and providing declarative control over AI tools and agents.

6. **Observability & Insight**  
   First-class analytics, logging, and reporting.  
   - Native support for **OpenTelemetry**, **Prometheus**, **Grafana**, and **Elastic Stack** for metrics and traces.  
   - Pluggable exporters so users can stream data to tools like **Datadog**, **New Relic**, or **OpenSearch**.  
   - Built-in reports and visual insights accessible from both the TUI and Web UI.

7. **Embeddable SDK**  
   Use BurstGridGo as a library within Go projects to orchestrate internal tasks or integrate directly into CI/CD and infrastructure workflows.

## 🚀 Getting Involved

Right now, BurstGridGo is focused on **contributors**.  
If you want to explore, learn, or help me shape the engine as it evolves — start here:

👉 **Read the [Contributing Guide](CONTRIBUTING.md)** to understand how the project is structured, how to run it locally, and where your input can make the biggest difference.

This stage is about **building the foundation** — architecture, testing, documentation, and modules.
Every contribution helps move BurstGridGo closer to becoming the platform I envision.

## 📚 Documentation

- **[Architecture Overview](docs/architecture.md)** - Comprehensive diagrams showing the system architecture, package dependencies, and execution pipeline
- **[Project Guide (CLAUDE.md)](CLAUDE.md)** - Development guide with commands, testing strategy, and architectural decisions
- **[Contributing Guide](CONTRIBUTING.md)** - How to contribute to the project

## License
burstgridgo is licensed under the MIT License.
