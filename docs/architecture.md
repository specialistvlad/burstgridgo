# BurstGridGo Architecture

This document provides architectural diagrams showing the structure and relationships of all packages in BurstGridGo.

## Table of Contents
- [High-Level Architecture](#high-level-architecture)
- [Package Dependency Graph](#package-dependency-graph)
- [Execution Pipeline](#execution-pipeline)
- [Package Descriptions](#package-descriptions)

---

## High-Level Architecture

The system follows a **5-phase declarative execution pipeline**:

```mermaid
graph TB
    subgraph "Phase 1: Parsing"
        CLI[cmd/cli<br/>CLI Entry Point]
        App[internal/app<br/>Application]
        HCL[internal/bggohcl<br/>HCL Parser]
        Model[internal/model<br/>Grid/Runner/Step]
        CLI --> App
        App --> HCL
        HCL --> Model
    end

    subgraph "Phase 2: Expression Analysis"
        Expr[internal/bggoexpr<br/>Expression Evaluator]
        Model --> Expr
    end

    subgraph "Phase 3: Graph Construction"
        Session[internal/session<br/>Session Interface]
        LocalSession[internal/localsession<br/>Local Implementation]
        Builder[internal/builder<br/>Task Builder]
        Graph[internal/graph<br/>DAG Graph]
        Topo[internal/inmemorytopology<br/>Topology Store]
        Store[internal/inmemorystore<br/>Node State Store]

        App --> Session
        Session --> LocalSession
        LocalSession --> Graph
        Graph --> Topo
        Graph --> Store
        LocalSession --> Builder
    end

    subgraph "Phase 4: Scheduling"
        Scheduler[internal/scheduler<br/>Dependency Scheduler]
        LocalSession --> Scheduler
        Scheduler --> Graph
    end

    subgraph "Phase 5: Execution"
        Executor[internal/executor<br/>Executor Interface]
        LocalExec[internal/localexecutor<br/>Local Executor]
        Task[internal/task<br/>Runnable Task]
        Handlers[internal/handlers<br/>Handler Registry]
        Registry[internal/registry<br/>Module Registry]
        Modules[modules/*<br/>Module Implementations]

        LocalSession --> Executor
        Executor --> LocalExec
        LocalExec --> Scheduler
        LocalExec --> Builder
        Builder --> Task
        LocalExec --> Handlers
        Registry --> Handlers
        Modules --> Registry
    end

    subgraph "Supporting Utilities"
        NodeID[internal/nodeid<br/>Node Addressing]
        Node[internal/node<br/>Node Definition]
        CtxLog[internal/ctxlog<br/>Context Logging]
        FSUtil[internal/fsutil<br/>File Utilities]

        Graph --> NodeID
        Graph --> Node
        App --> CtxLog
        App --> FSUtil
    end

    style CLI fill:#e1f5ff
    style App fill:#e1f5ff
    style Model fill:#fff3e0
    style Expr fill:#fff3e0
    style Graph fill:#f3e5f5
    style Session fill:#f3e5f5
    style Executor fill:#e8f5e9
    style LocalExec fill:#e8f5e9
    style Modules fill:#ffe0b2
```

---

## Package Dependency Graph

Detailed view of all internal packages and their dependencies:

```mermaid
graph LR
    subgraph "Entry Point"
        CLI[cmd/cli]
    end

    subgraph "Application Layer"
        App[internal/app]
        Config[internal/app/config]
        CtxLog[internal/ctxlog]
    end

    subgraph "Parsing & Model"
        Model[internal/model]
        BGGOHCL[internal/bggohcl]
        BGGOExpr[internal/bggoexpr]
        FSUtil[internal/fsutil]
    end

    subgraph "Session Management"
        Session[internal/session]
        LocalSession[internal/localsession]
    end

    subgraph "Graph & Topology"
        Graph[internal/graph]
        InMemoryTopology[internal/inmemorytopology]
        InMemoryStore[internal/inmemorystore]
        TopologyStore[internal/topologystore]
        NodeStore[internal/nodestore]
        Node[internal/node]
        NodeID[internal/nodeid]
    end

    subgraph "Execution Engine"
        Executor[internal/executor]
        LocalExecutor[internal/localexecutor]
        Scheduler[internal/scheduler]
        Builder[internal/builder]
        Task[internal/task]
    end

    subgraph "Module System"
        Registry[internal/registry]
        Handlers[internal/handlers]
        ModulePrint[modules/print]
    end

    %% CLI dependencies
    CLI --> App
    CLI --> internal/cli

    %% App dependencies
    App --> CtxLog
    App --> Model
    App --> Registry
    App --> Session
    App --> LocalSession
    App --> FSUtil

    %% Session dependencies
    Session --> Executor
    LocalSession --> Session
    LocalSession --> Graph
    LocalSession --> Builder
    LocalSession --> Scheduler
    LocalSession --> LocalExecutor
    LocalSession --> InMemoryTopology
    LocalSession --> InMemoryStore

    %% Executor dependencies
    Executor --> Node
    LocalExecutor --> Executor
    LocalExecutor --> Scheduler
    LocalExecutor --> Builder
    LocalExecutor --> Graph
    LocalExecutor --> Handlers

    %% Scheduler dependencies
    Scheduler --> Graph
    Scheduler --> Node
    Scheduler --> CtxLog

    %% Builder dependencies
    Builder --> Node
    Builder --> Graph
    Builder --> Task
    Builder --> CtxLog

    %% Graph dependencies
    Graph --> TopologyStore
    Graph --> NodeStore
    Graph --> Node
    Graph --> NodeID
    InMemoryTopology --> TopologyStore
    InMemoryStore --> NodeStore

    %% Task dependencies
    Task --> Node

    %% Node dependencies
    Node --> NodeID

    %% Model dependencies
    Model --> BGGOHCL
    Model --> BGGOExpr

    %% Registry dependencies
    Registry --> Handlers
    Registry --> Model
    ModulePrint --> Registry

    %% Handlers dependencies
    Handlers -.-> Registry

    style CLI fill:#e1f5ff,stroke:#01579b,stroke-width:3px
    style App fill:#e1f5ff,stroke:#0277bd
    style Model fill:#fff3e0,stroke:#e65100
    style Session fill:#f3e5f5,stroke:#4a148c
    style LocalSession fill:#f3e5f5,stroke:#6a1b9a
    style Graph fill:#f3e5f5,stroke:#7b1fa2
    style Executor fill:#e8f5e9,stroke:#1b5e20
    style LocalExecutor fill:#e8f5e9,stroke:#2e7d32
    style Registry fill:#ffe0b2,stroke:#e65100
    style Handlers fill:#ffe0b2,stroke:#ef6c00
    style ModulePrint fill:#ffe0b2,stroke:#f57c00
```

---

## Execution Pipeline

Step-by-step flow from HCL file to execution:

```mermaid
sequenceDiagram
    participant User
    participant CLI as cmd/cli
    participant App as internal/app
    participant Model as internal/model
    participant Session as internal/localsession
    participant Executor as internal/localexecutor
    participant Scheduler as internal/scheduler
    participant Builder as internal/builder
    participant Handler as modules/*/handler
    participant Graph as internal/graph

    User->>CLI: make run example.hcl
    CLI->>App: NewApp(config)
    CLI->>App: LoadModules()
    App->>Registry: Register modules
    CLI->>App: LoadGrids()
    App->>Model: ParseHCL(file)
    Model-->>App: Grid (parsed model)

    CLI->>App: Run()
    App->>Session: NewSession(grid, handlers)
    Session->>Graph: New(topology, store)
    Session->>Scheduler: New(graph)
    Session->>Builder: New()
    Session->>Executor: New(scheduler, graph, builder, handlers)
    Session-->>App: session

    App->>Session: GetExecutor()
    Session-->>App: executor

    App->>Executor: Execute(ctx)

    Note over Executor: Phase 1: Build Graph from Model
    Executor->>Graph: AddNode(step1)
    Executor->>Graph: AddNode(step2)
    Executor->>Graph: AddDependency(step2 → step1)

    Note over Executor: Phase 2: Execution Loop
    loop For each ready node
        Executor->>Scheduler: ReadyNodes()
        Scheduler->>Graph: GetStatus(nodes)
        Scheduler-->>Executor: node (channel)

        Executor->>Graph: MarkRunning(node.ID)

        Executor->>Builder: Build(node, graph)
        Note over Builder: Resolve expressions<br/>Prepare inputs
        Builder-->>Executor: task

        Executor->>Handler: Execute(task.ResolvedInputs)
        Handler-->>Executor: output / error

        alt Success
            Executor->>Graph: SetOutput(node.ID, output)
            Executor->>Graph: MarkCompleted(node.ID)
        else Failure
            Executor->>Graph: MarkFailed(node.ID, error)
        end
    end

    Executor-->>App: nil / error
    App-->>CLI: exit code
    CLI-->>User: result
```

---

## Package Descriptions

### Entry Point & Application
- **cmd/cli** - CLI entry point, argument parsing via `internal/cli`, panic recovery
- **internal/app** - Application initialization, module/grid loading, session orchestration
- **internal/cli** - CLI argument parsing and validation
- **internal/ctxlog** - Context-aware structured logging using `slog`

### Parsing & Model Layer
- **internal/model** - Core data structures: `Grid`, `Runner`, `Step`, `Variable`, `Local`
- **internal/bggohcl** - HCL parsing utilities using HashiCorp HCL v2
- **internal/bggoexpr** - Expression extraction and analysis using go-cty
- **internal/fsutil** - File system utilities for finding HCL files

### Session Management
- **internal/session** - Session abstraction (interface for local/distributed execution)
- **internal/localsession** - Local session implementation with dependency injection

### Graph & State Management
- **internal/graph** - Stateful DAG representation, node state tracking
- **internal/node** - Node definition (ID, type, config, dependencies, status)
- **internal/nodeid** - Node addressing system (`<type>.<name>[index]`)
- **internal/inmemorytopology** - In-memory topology storage (DAG structure)
- **internal/inmemorystore** - In-memory node state storage (status, output, errors)
- **internal/topologystore** - Topology store interface
- **internal/nodestore** - Node store interface

### Execution Engine
- **internal/executor** - Executor interface (DAG execution orchestration)
- **internal/localexecutor** - Local executor implementation
- **internal/scheduler** - Dependency-based node scheduling
- **internal/builder** - Task builder (expression resolution, input preparation)
- **internal/task** - Runnable task representation

### Module System
- **internal/registry** - Module registry, runner definitions, handler management
- **internal/handlers** - Handler registration and lookup
- **modules/print** - Example module (print runner implementation)

### Testing & Utilities
- **internal/integrationtests** - End-to-end integration tests (67.5% coverage)
- **internal/testutil** - Testing utilities and harnesses

---

## Architecture Highlights

### 1. Layered Architecture
Clean separation between parsing, validation, graph building, and execution phases.

### 2. Interface-Driven Design
- `session.Session` - Execution environment abstraction
- `executor.Executor` - Execution orchestration interface
- `scheduler.Scheduler` - Node scheduling interface
- `builder.Builder` - Task building interface
- `graph.Graph` - Graph operations interface

### 3. Store Pattern
State and topology managed via separate store interfaces for flexibility:
- Topology (DAG structure) → `topologystore.TopologyStore`
- Node state (status, output) → `nodestore.NodeStore`

### 4. Type-Safe Throughout
HCL → strongly-typed Go structs → validated model → executable graph

### 5. Expression Isolation
HCL expressions evaluated during graph construction, not during execution.

### 6. Dependency Injection
Session factory wires up all execution components with proper dependencies.

---

## Current Implementation Status

| Layer | Status | Notes |
|-------|--------|-------|
| **CLI & App** | ✅ Complete | Argument parsing, logging, configuration |
| **HCL Parsing** | ✅ Complete | Fully parses HCL into model structs |
| **Model Layer** | ✅ Complete | Grid, Runner, Step structures |
| **Expression Analysis** | ✅ Complete | Extracts references and functions |
| **Node Addressing** | ✅ Complete | Robust `<type>.<name>[index]` system |
| **Session** | ✅ Wired | Creates and wires dependencies |
| **Graph** | ⚠️ Defined | Interfaces defined, needs population from model |
| **Scheduler** | ❌ Stub | Returns empty channel |
| **Builder** | ❌ Stub | Returns empty task |
| **Executor** | ❌ Stub | Just logs "placeholder" |
| **Handlers** | ✅ Complete | Registration works |
| **Modules** | ⚠️ Minimal | Only `print` module exists |

**Next Priority:** Implement execution pipeline (graph population → scheduler → builder → executor loop)

---

Generated: 2025-11-21