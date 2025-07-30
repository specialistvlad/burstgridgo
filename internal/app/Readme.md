# Internal Architecture

This document provides a deep look into the internal architecture of `burstgridgo`. Understanding these concepts is essential for contributing new features or modules.

The architecture is designed to be **declarative, extensible, and type-safe**. It cleanly separates the **definition** of a component from its Go **implementation** and its user-facing **instance**.

<br>

## Core Concepts: Actions vs. Assets
The engine is built on a fundamental duality: stateless **Actions** and stateful **Assets**. This separation allows for transient tasks (like making a single API call) to safely use long-lived, shared resources (like a database connection pool).

| Concept | The Blueprint | The Go Implementation | The User's Instance |
| :--- | :--- | :--- | :--- |
| **Stateless Action** | `runner` | `RegisteredRunner` | `step` |
| **Stateful Asset** | `asset` | `RegisteredAsset` | `resource` |

<br>

## Application Lifecycle & Core Packages

The system is composed of several interlocking packages that manage the application's lifecycle from startup to shutdown.

**Package Responsibility Diagram (Text Description):**

* **1. Foundational Packages:**
    * **`schema`**: Defines all the core HCL data structures (`Step`, `Resource`, `RunnerDefinition`, etc.). It has no other internal dependencies.
    * **`registry`**: Provides the central `Registry` struct that holds mappings between HCL definitions and their Go implementations. Modules use this to register their functionality.

* **2. Module Author Defines:**
    * **Manifest File** (`.hcl`): Defines the public API of a 'runner' or 'asset' using the structures from the `schema` package.
    * **Go `Module`** (`.go`): Implements the component's logic in Go and provides a `Register` method to add its handlers to the `registry`.

* **3. User Defines:**
    * **User Grid File** (`.hcl`): Creates instances ('steps' and 'resources') of the components defined by the module author to describe a workflow.

* **4. Engine on Run (Core Packages):**
    * **`app`**: The main application orchestrator. It handles configuration, discovers and registers all modules, and drives the overall execution flow.
    * **`dag`**: The graph-building layer. It takes the user's configuration and builds a validated execution graph, resolving dependencies and detecting cycles. **It does not run the graph.**
    * **`executor`**: The execution layer. It takes a pre-built graph from the `dag` package and manages its concurrent execution using a worker pool.

### 1. The Manifest File (`*.hcl`)
A manifest is an HCL file that defines the public API of a `runner` or an `asset`. It specifies the component's type, its inputs, its outputs, and which Go functions implement its lifecycle. These live inside each module's directory.
* **`runner "type" {}`**: Defines a stateless action. Its key lifecycle event is `on_run`.
* **`asset "type" {}`**: Defines a stateful asset. Its key lifecycle events are `create` and `destroy`.
* **`input "name" {}`**: Defines an input argument, its type, and an optional `default` value.
* **`uses "local_name" {}`**: Declares a dependency on an `asset` type, enabling dependency injection for a runner.

### 2. The Go Handler (`module.go`)
This is the Go code that implements the component's logic, exposed via a type that satisfies the `registry.Module` interface.
* **Structs for I/O**: Go structs with `hcl` tags are used to define inputs (`Input`) and injected dependencies (`Deps`). This provides compile-time type safety for the handler author.
* **Enforced Contract**: The application performs a strict validation check at startup to ensure that the `input` blocks in the manifest and the `hcl` tags in the Go `Input` struct are perfectly in sync. The application will fail to start with a descriptive error if there are any discrepancies.
* **Standardized Handler Signatures**:
    * Runner: `func(ctx context.Context, deps any, input any) (cty.Value, error)`
    * Asset `Create`: `func(ctx context.Context, input any) (any, error)`
    * Asset `Destroy`: `func(resource any) error`
* **Registration**: In the `app` startup sequence, each module's `Register` method is called. This method calls `registry.RegisterRunner` or `registry.RegisterAssetHandler`, linking the string name from the manifest's `lifecycle` block to the actual Go function.

### 3. The User's Grid File (`*.hcl`)
This is where a user defines a workflow by creating instances of runners and assets.
* **`step "type" "name" {}`**: Creates an executable instance of a `runner`.
* **`resource "type" "name" {}`**: Creates a managed, stateful instance of an `asset`.
* **`arguments {}`**: Provides the concrete values for the `input`s defined in the manifest.
* **`uses {}`**: Injects a live `resource` instance into a `step`. This is the mechanism for sharing resources.

<br>

## Execution Flow

**Execution Flow (Step-by-Step):**

* **Step 1: Startup & Validation**
    * The `app.NewApp` constructor is called. It creates a new `registry.Registry`.
    * It discovers and registers all Go modules (`registry.Module`) and then discovers and parses all HCL manifest files from the configured modules path.
    * A strict validation check (`registry.ValidateRegistry`) is run to ensure HCL definitions and Go implementations are in sync. If not, startup is aborted with an error.

* **Step 2: Graph Construction**
    * The `app.Run` method is called. It first loads and parses the user's grid file(s).
    * It then calls `dag.Build`, passing it the grid configuration and the populated registry. The `dag` package creates a node for each `step` and `resource`, links all dependencies (both explicit and implicit), and returns a validated, ready-to-run graph.

* **Step 3: Execution**
    * `app.Run` creates a new `executor.Executor`, giving it the graph, worker count, and registry.
    * `executor.Run` is called. It identifies all root nodes (those with no dependencies) and feeds them into a channel for the worker pool.
    * As workers complete nodes, they atomically decrement the dependency counters of downstream nodes. When a node's counter reaches zero, it is unlocked and scheduled for execution.

* **Step 4: Cleanup**
    * The `executor` tracks all created resources. As soon as a resource is no longer needed by any running or pending steps, its `Destroy` handler is scheduled for efficient cleanup.
    * Any remaining resources are guaranteed to be destroyed when `executor.Run` completes.

<br>

## Further Reading
For a more detailed technical breakdown of each component, please see the package-level documentation (the `doc.go` file) within each of the subdirectories: `app`, `cli`, `ctxlog`, `dag`, `executor`, `registry`, and `schema`.