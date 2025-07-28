# Architecture

This document provides a deep look into the internal architecture of `burstgridgo`. Understanding these concepts is essential for contributing new features or modules.

The architecture is designed to be **declarative, extensible, and type-safe**. It cleanly separates the **definition** of a component from its Go **implementation** and its user-facing **instance**.

<br>

## Core Concepts: Actions vs. Assets
The engine is built on a fundamental duality: stateless **Actions** and stateful **Assets**. This separation allows for transient tasks (like making a single API call) to safely use long-lived, shared resources (like a database connection pool).

| Concept | The Blueprint | The Go Implementation | The User's Instance |
| :--- | :--- | :--- | :--- |
| **Stateless Action** | `runner` | `RegisteredHandler` | `step` |
| **Stateful Asset** | `asset` | `RegisteredAssetHandler` | `resource` |

<br>

## Key Components

The system is composed of four interlocking parts: the **Schema** (the data structures), the **Registry** (the registration mechanism), the **Go Handler** (the implementation), and the **Grid File** (the user's execution plan).

**Component Relationship Diagram (Text Description):**

* **1. Foundational Packages:**
    * **`schema` Package**: Defines all the core HCL data structures (`Step`, `Resource`, `RunnerDefinition`, etc.). It has no internal dependencies.
    * **`registry` Package**: Provides the global `HandlerRegistry` and `AssetHandlerRegistry` and the functions (`RegisterHandler`) that modules use to register their implementations.

* **2. Module Author Defines:**
    * **Manifest File** (`.hcl`): Defines the public API of a 'runner' or 'asset' using the structures from the `schema` package.
    * **Go Handler File** (`.go`): Implements the logic in Go and registers it with the `registry` package in its `init()` block.

* **3. User Defines:**
    * **User Grid File** (`.hcl`): Creates instances ('steps' and 'resources') of the components defined by the module author.

* **4. Engine on Run:**
    * **`engine` Package**: The loading layer. It discovers and parses all manifest and grid files.
    * **`dag` Package**: The execution layer. It builds a graph from the user's grid and executes it by connecting the user's instances to the Go handlers found in the `registry`.

### 1. The Manifest File (`*.hcl`)
A manifest is an HCL file that defines the public API of a `runner` or an `asset`. It specifies the component's type, its inputs, its outputs, and which Go functions implement its lifecycle. These live inside each module's directory.
* **`runner "type" {}`**: Defines a stateless action. Its key lifecycle event is `on_run`.
* **`asset "type" {}`**: Defines a stateful asset. Its key lifecycle events are `create` and `destroy`.
* **`input "name" {}`**: Defines an input argument, its type, default value, and whether it's optional.
* **`uses "local_name" {}`**: Declares a dependency on an `asset` type, enabling dependency injection.

### 2. The Go Handler (`module.go`)
This is the Go code that implements the component's logic.
* **Structs for I/O**: Go structs with `hcl` tags are used to define inputs (`Input`) and dependencies (`Deps`). This provides compile-time type safety for the handler author.
* **Standardized Handler Signatures**:
    * Runner: `func(ctx, deps *Deps, input *Input) (cty.Value, error)`
    * Asset `Create`: `func(ctx, input *Input) (any, error)`
    * Asset `Destroy`: `func(resource any) error`
* **Registration**: In an `init()` block, the Go handler is registered with the central `registry.RegisterHandler`. This links the string name from the manifest's `lifecycle` block to the actual Go function.

### 3. The User's Grid File (`*.hcl`)
This is where a user defines a workflow by creating instances of runners and assets.
* **`step "type" "name" {}`**: Creates an executable instance of a `runner`.
* **`resource "type" "name" {}`**: Creates a managed, stateful instance of an `asset`.
* **`arguments {}`**: Provides the concrete values for the `input`s defined in the manifest.
* **`uses {}`**: Injects a live `resource` instance into a `step`. This is the mechanism for sharing resources.

<br>

## Execution Flow

**Execution Flow (Step-by-Step):**

* **Step 1: Discovery & Registration (Startup)**
    * The `engine` scans the `modules/` directory for all HCL manifest files and populates the `DefinitionRegistry` (which lives in the `registry` package).
    * Simultaneously, Go's runtime executes `init()` functions in the Go modules, populating the `HandlerRegistry` in the `registry` package.

* **Step 2: Grid Parsing**
    * The `engine` loads and merges the user's grid file(s) into a single `GridConfig` object.

* **Step 3: DAG Construction**
    * The `dag` package creates a graph node for each `step` and `resource`.
    * Dependencies are calculated from variable usage, `depends_on`, and `uses` blocks to create the graph edges.
    * A cycle check is performed to ensure the graph is valid.

* **Step 4: Execution**
    * The `app` layer creates an `Executor`, passing it the populated registries from the `registry` package.
    * The executor's worker pool begins processing nodes that have no pending dependencies.
    * As nodes complete, their dependent nodes are unlocked and scheduled for execution.

For a more detailed technical breakdown of the engine and DAG packages, see their internal README files:
* [internal/engine/README.md](../internal/engine/README.md)
* [internal/dag/README.md](../internal/dag/README.md)