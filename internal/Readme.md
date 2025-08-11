# Internal Architecture & Data Flow

This document provides a deep look into the internal architecture of `burstgridgo`. Understanding these concepts is essential for contributing new features or modules.

---

## Part 1: Internal Architecture

The architecture is designed to be **declarative, extensible, and type-safe**. It cleanly separates the **definition** of a component from its Go **implementation** and its user-facing **instance**.

<br>

### Core Concepts: Actions vs. Assets

The engine is built on a fundamental duality: stateless **Actions** and stateful **Assets**. This separation allows for transient tasks (like making a single API call) to safely use long-lived, shared resources (like a database connection pool).

| Concept          | The Blueprint      | The Go Implementation | The User's Instance |
| :--------------- | :----------------- | :-------------------- | :------------------ |
| **Stateless Action** | `runner`           | `RegisteredRunner`    | `step`              |
| **Stateful Asset** | `asset`            | `RegisteredAsset`     | `resource`          |

<br>

### Application Lifecycle & Core Packages

The system is composed of several interlocking packages that manage the application's lifecycle from startup to shutdown.

**Package Responsibility Diagram (Text Description):**

* **1. Foundational Packages:**
    * **`registry`**: Provides the central `Registry` struct that holds mappings between definition names and their Go implementations. Modules use this to register their functionality.
    * **`nodeid`**: Provides a structured, type-safe representation for node identifiers (`nodeid.Address`). It centralizes all parsing, formatting, and validation of IDs (e.g., `step.http_client.get_user[0]`), eliminating fragile string manipulation throughout the rest of the application.
    * **`node`**: Defines the `node.Node` struct, the core data structure representing a single vertex in the execution graph. It encapsulates a node's configuration, its structured ID (`nodeid.Address`), its current state, and its execution result.
    * **`dag`**: Provides a **generic, thread-safe implementation** of a Directed Acyclic Graph. It is only concerned with the graph's topology (the relationships between vertices) and operates on simple `string` identifiers. It provides core functionalities like adding nodes/edges and cycle detection.

* **2. Module Author Defines:**
    * **Manifest File** (`.hcl`): Defines the public API of a 'runner' or 'asset'.
    * **Go `Module`** (`.go`): Implements the component's business logic in pure Go.
        * **Inputs:** The module's `Input` **must** be a pure Go `struct`. **Fields in this top-level struct use the `bggo:"..."` tag to map HCL arguments. Any nested structs representing HCL objects must use the `cty:"..."` tag for their fields.**
        * **Outputs:** If a handler returns data, it **must** be a pure Go `struct` that uses `cty:"..."` tags on its fields to map them to the attribute names expected by the engine.

* **3. User Defines:**
    * **User Grid File** (`.hcl`): Creates instances ('steps' and 'resources') of the components defined by the module author to describe a workflow.

* **4. Engine on Run (Core Packages):**
    * **`config`**: Defines the **format-agnostic** configuration `Model` and the core `Loader` and `Converter` interfaces. This is the pure data model the rest of the engine works with.
    * **`hcl`**: Provides the concrete **HCL implementation** of the `config.Loader` and `config.Converter` interfaces. It's responsible for all file parsing and HCL-specific data binding.
    * **`app`**: The main application orchestrator. It initializes a `config.Loader` (e.g., the `hcl.Loader`), drives the loading process, and manages the overall execution flow.
    * **`builder`**: The graph construction layer. It takes the format-agnostic `config.Model`, creates a `node.Node` for each step/resource (each with a structured `nodeid.Address`), and uses the `dag` package to assemble them into a final, validated execution graph.
    * **`executor`**: The execution layer. It takes a pre-built graph from the `builder` and manages its concurrent execution. It uses the `config.Converter` to perform just-in-time data binding.

---

## Part 2: Data Flow and Validation Pipeline

This section provides a definitive, end-to-end description of how data flows through the `burstgridgo` application. It details the validation stages from the moment the application starts to the final execution of a module's business logic.

### Phase 1: Application Startup & Static Validation

This phase occurs **once** each time the `burstgridgo` application is launched. Its primary purpose is to load all configuration files and perform critical, upfront checks to ensure the structural integrity of all registered modules. This guarantees that the application is in a valid state before any execution begins.

The sequence is as follows:

1.  **Unified Configuration Loading**:
    * The application gathers all user-provided configuration paths (e.g., from `--grid` and `--modules-path` flags).
    * These paths are passed to a `config.Loader` (currently the `hcl.Loader` implementation).
    * The loader recursively discovers all `.hcl` files, parses their contents (identifying `runner`, `asset`, `step`, and `resource` blocks), and assembles them into a single, in-memory `config.Model`. This model represents the entire desired state for the run.

2.  **Module Registration & Parity Validation**:
    * This is a two-part static validation step:
        * **Handler-Manifest Linking**: The engine ensures that every lifecycle handler named in a manifest (e.g., `lifecycle { on_run = "OnRunMyModule" }`) corresponds to a Go handler that was actually registered in the `registry`.
        * **Input/Type Parity**: The engine performs a strict parity check (`registry.ValidateRegistry()`) between the manifest and the Go `Input` struct. This validation is twofold:
            * **Presence**: It ensures every `input` block in the manifest has a corresponding `bggo:"..."` tagged field in the Go struct, and vice-versa.
            * **Type**: It ensures the `type` declared in the manifest (e.g., `type = list(number)`) is compatible with the type of the Go field (e.g., `[]int`). **This validation is fully recursive, checking the entire shape and all attribute types of nested objects.** If they are not compatible, the application will fail to start.

### Phase 2: Graph Construction

This phase is orchestrated by the `builder` package and bridges the gap between the static configuration model and the executable graph.

1.  **Node Instantiation**: The `builder` iterates over all `step` and `resource` blocks in the `config.Model`. For each, it creates a corresponding `node.Node` struct.
2.  **ID Assignment (`ADR-014`)**: During instantiation, each `node.Node` is assigned a unique, structured `nodeid.Address`. This ensures that from the moment of its creation, every node has a valid, parseable identifier.
3.  **Dependency Linking**: The `builder` inspects each node's `depends_on` block (for explicit dependencies) and all HCL expressions in its arguments (for implicit dependencies, like `${step.A.output}`). It translates these references into directed edges in the graph, using the generic `dag` package to manage the topology.
4.  **Topological Validation**: Once all nodes and edges are in place, the `builder` calls the `dag` package's cycle detection algorithm. If a cycle is found (e.g., A depends on B, and B depends on A), the application fails to start with a clear error.

### Phase 3: Per-Step Runtime Pipeline

This pipeline is executed by the `executor` for **every `step` node** in the directed acyclic graph (DAG) during a run. It describes the "just-in-time" process of preparing data for, validating, and executing a module's business logic.

The pipeline ensures that a module handler receives a pure, simple Go struct containing data that has been thoroughly vetted.

The sequence is as follows:

1.  **Expression Evaluation**:
    * The `executor` first identifies any HCL expressions within the step's argument block (e.g., `args = { message = "Hello, ${step.A.output.name}!" }`).
    * It resolves these expressions using the current evaluation context, substituting them with their real, calculated values.

2.  **Default Value Application**:
    * Before decoding user-provided arguments, the `executor` checks the module's manifest for any inputs that have a `default` value.
    * If a user omits an optional argument that has a defined default, the engine applies that default value, ensuring predictable behavior.

3.  **Type System & Validation (ADR-009, ADR-010, & ADR-011)**:
    * The engine now uses the `type` from the manifest as the single source of truth. The following types are supported:
        * **Primitives:** `string`, `number`, `bool`
        * **Collections:** `list(T)`, `map(T)`, `set(T)` where `T` is one of the primitive types.
        * **Objects:**
            * **`object({key=type, ...})`**: A structurally-typed object that maps to a Go `struct`.
            * **`object({})`**: A generic object that maps to a Go `map[string]any`.
    * The engine attempts to convert the user-provided value (or the default value) to this declared type.
    * If the conversion fails (e.g., passing `["a", 1]` to an input of type `list(string)`), the run fails immediately with a clear type-mismatch error.

4.  **Input Translation (`ADR-008`)**:
    * The `executor` creates a new, zero-value instance of the module's pure Go `Input` struct.
    * It uses the `config.Converter` interface to decode the type-validated data into the Go struct, using the `bggo:"..."` tags for mapping.

5.  **Execution**:
    * Only after all preceding stages does the `executor` finally call the module's `OnRun(ctx, input)` method.
    * The handler receives the fully populated, translated, and validated pure Go `Input` struct, allowing the author to focus entirely on the module's business logic.

6.  **Output Translation (`ADR-008`)**:
    * After the pure Go handler executes and returns its `Output` struct, the `executor` calls the `converter` again.
    * The converter translates this native struct back into the engine's internal representation (e.g., a `cty.Value` object). It inspects the `cty:"..."` tags on the struct's fields to ensure the output can be correctly used by downstream steps that depend on it.

### Future Work

* **`ADR-012` (Planned)**: Introduce support for **meta-arguments** (`count`, `for_each`) to allow dynamic creation of multiple step and resource instances.
* Declarative features like `validation {}` blocks and sensitive input handling will be built on top of this type system in subsequent ADRs.