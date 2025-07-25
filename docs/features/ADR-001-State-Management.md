# ADR-001: Stateful Resource Management
- **Status**: Implemented
- **Author**: Vladyslav Kazantsev
- **Date**: 2025-07-24

This document outlines the architecture for introducing stateful, managed assets into the workflow engine. This design enables resource sharing (e.g., connection pools, authenticated clients) between steps in a safe, efficient, and declarative manner.

---
## 1. Core Concepts & Naming Convention

The system distinguishes between two fundamental concepts: stateless **Actions** and stateful **Assets**. This proposal introduces the Asset concept to work alongside the existing Action primitives.

| Concept | In Module Manifest (The Blueprint) | In User's HCL (The Instance) |
| :--- | :--- | :--- |
| **The Action** | **`runner`**: Defines a stateless, runnable action. | **`step`**: Executes an instance of a `runner`. |
| **The Asset** | **`asset`**: Defines the schema for a stateful, shareable object. | **`resource`**: Creates a managed instance of an `asset`. |

---
## 2. Lifecycles & Responsibilities

The engine treats `resources` and `steps` as distinct node types within the execution graph. A `resource` **is** a thing, while a `step` **does** a thing.

#### **`resource` Lifecycle: `Create` & `Destroy`**
A **`resource`** is a **long-lived, stateful asset**. Its lifecycle is managed by two main methods implemented in its Go handler:

* **`Create()`**: Called once by the executor when the `resource` is first needed. Its purpose is to initialize the asset (e.g., create a database connection pool, authenticate with an API) and return the stateful Go object.
* **`Destroy()`**: Called by the executor to release the asset. Its execution is guaranteed if `Create()` was successfully called. The trigger for this is twofold:
   1. **Efficiently during a run**: The executor schedules `Destroy()` as soon as all downstream `steps` that depend on the `resource` have completed. This is managed by a **descendant completion counter** on the `resource` node itself.
   2. **Reliably upon exit**: A deferred **cleanup stack** in the executor guarantees that `Destroy()` is called for all created resources when the grid run terminates, regardless of success or failure.
A `resource` does not have an `on_run` method because its job is to *exist* and provide a persistent capability, not to perform a transient action.

#### **`step` Lifecycle: `on_run`, `on_start`, & `on_end`**
A **`step`** is a **transient, stateless action**. Its primary lifecycle method is:

* **`on_run()`**: The main body of the `step`, containing the business logic to perform a single, discrete action.

Optional hooks provide guaranteed, isolated cleanup for the action itself:

* **`on_start()`**: An optional method that runs immediately before `on_run`.
* **`on_end()`**: An optional method that runs immediately after `on_start` has been called, even if `on_run` fails. This is crucial for action-specific setup/teardown, like managing a temporary file or a distributed lock.

---
## 3. The Provider Contract: Dependency Injection via `uses` block
To establish a safe and explicit link between a `step` and the `resource`(s) it needs, a `step` block will use a dedicated `uses {}` block. This block maps resource instances directly to the fields of a Go struct that is passed to the handler, providing compile-time type safety for the handler author.

#### HCL Contract
A `runner`'s manifest will still declare the *types* of assets it is compatible with. The user's `step` block provides the concrete `resource` *instances* via the `uses {}` block.

* **Runner Manifest (`modules/sql/manifest.hcl`)**
    This file declares the `asset` types this runner can consume.

    ```hcl
    runner "sql_query" {
      description = "Executes a SQL query against a database resource."
      
      # This runner declares it needs a dependency that conforms 
      # to the "database" asset type. The key "DB" is a suggestion
      # for the field name in the Go Deps struct.
      uses "DB" {
        asset_type = "database"
      }

      input "query" { 
        type        = string
        description = "The SQL query to execute."
      }
    }
    ```

* **User Grid File (`my_test.hcl`)**
    The user maps a specific `resource` instance to the dependency key defined in the manifest.

    ```hcl
      resource "database" "main_db" {
        connection_string = "postgres://user:pass@host:port/db"
      }

      step "sql_query" "get_users" {
        # The step explicitly maps the 'main_db' resource to the 'DB' dependency.
        # The engine uses this mapping to populate the Go handler's Deps struct.
        uses {
          DB = resource.database.main_db
        }

        arguments {
          query = "SELECT * FROM users;"
        }
      }
    ```

#### Go Handler Contract
The Go handler for the `step` receives its dependencies in a dedicated, type-safe `Deps` struct. The handler signature is standardized to `func(ctx context.Context, deps *Deps, input *Input) (any, error)`.

* **Go Handler Implementation (`modules/sql/module.go`)**

    ```golang
    package sql

    // --- In a shared package, e.g., assets/database ---
    // The interface defining the contract for any "database" resource.
    type Connection interface {
        Exec(ctx context.Context, query string, args ...any) (any, error)
        Close() error
    }

    // --- In the sql_query runner's package ---

    // Deps defines the resources required by this handler.
    // Field names must match the keys in the HCL `uses` block.
    type Deps struct {
        DB database.Connection
    }

    // Input defines the arguments for the 'arguments' block.
    type Input struct {
        Query string `hcl:"query"`
    }

    // OnRunSqlQuery is the handler for the step. Its signature is standardized.
    // The 'deps' argument is a pointer to a populated, type-safe struct.
    func OnRunSqlQuery(ctx context.Context, deps *Deps, input *Input) (any, error) {
        // Implementation has full type safety.
        result, err := deps.DB.Exec(ctx, input.Query)
        // ...
        return result, err
    }
    ```

---
## 4. Asset Type Registration & Validation

To bridge the gap between an HCL asset type (e.g., `"database"`) and a Go interface (`database.Connection`), and to enable the type-safe injection shown above, the engine will maintain a registry of asset interfaces.

Each `asset` implementation is responsible for registering the Go interface that its created objects will satisfy.

#### **Asset Interface Registry**
The engine will expose a new registration function:
```golang
// engine/asset.go
func RegisterAssetInterface(assetType string, iface reflect.Type)
```
Asset authors must call this in their package's `init()` block. This creates an explicit, machine-checkable link between the asset type string and its Go interface contract.

```golang
// In the database asset's Go package (e.g., assets/database/):
func init() {
    // This line tells the engine that any resource of type "database"
    // is expected to implement the database.Connection interface.
    engine.RegisterAssetInterface("database", reflect.TypeOf((*Connection)(nil)).Elem())
}
```
#### **Engine Validation Logic**
Before executing a `step`, the executor will perform the following validation:

1.  It identifies the required uses asset type from the `runner` manifest (e.g., `"database"`).
2.  It looks up the corresponding Go interface type in the `AssetInterfaceRegistry`.
3.  It retrieves the concrete `resource` object instance from the central store.
4.  It uses reflection to verify that the concrete object's type implements the required interface (`reflect.TypeOf(obj).Implements(iface)`).

If this check fails, the grid run is terminated with a clear type-mismatch error. This prevents runtime panics from failed type assertions and provides developers with immediate, precise feedback on configuration or implementation errors.

---
## 5. Engine Execution Plan

The engine's execution process is updated to manage both `resource` and `step` nodes within a single, unified DAG.

1.  **Unified DAG Construction:** The engine parses all HCL files and builds a single graph containing both `resource` and `step` nodes. A dependency edge is created from a `step` to a `resource` via the `uses` block, as well as from standard `step.A.output` interpolations.

2.  **Stateful Node Execution:** The executor's worker pool can execute any node whose dependencies are met.
    * When a worker executes a **`resource` node**, it calls the resource's `Create()` method. To ensure this happens exactly once and is concurrency-safe, the executor will use a `sync.Once` primitive for each resource node's creation logic. The stateful object returned by `Create()` is stored in a thread-safe central container, keyed by the resource's unique ID (e.g., `resource.database.main_db`).
    * When a worker executes a **`step` node**, the executor retrieves the required stateful uses object(s) from the central container, populates a Deps struct and passes a pointer to that struct to the handler

3.  **Resource Lifecycle Management & Cleanup:** The executor manages the resource lifecycle through a robust, two-part strategy to ensure both efficiency and reliability.
    * **Descendant Completion Counter (For Efficiency):** To enable efficient, runtime garbage collection, each `resource` node maintains an atomic counter.
        * *Initialization*: During graph construction, the executor performs a one-time traversal of the `resource`'s entire downstream dependency cone (all `step`s that depend on it, directly or transitively) and initializes the counter to the total number of these `step`s.
        * *Execution*: When a `step` completes (succeeds or fails), it atomically decrements the counter on each of its uses `resource`s.
        * *Trigger*: When a `resource`'s counter reaches zero, the executor immediately schedules its `Destroy()` method.

    * **Guaranteed Cleanup Stack (For Reliability):** To guarantee that resources are always released, the executor maintains an internal LIFO (Last-In, First-Out) stack of cleanup functions.
        * *Registration*: Immediately after a `resource`'s `Create()` method returns successfully, its `Destroy()` function is pushed onto this stack.
        * *Execution*: The main `executor.Run()` function uses a `defer` block that, upon any exit, unwinds this stack and executes each `Destroy()` function. This ensures cleanup even in the case of a panic or early termination.
