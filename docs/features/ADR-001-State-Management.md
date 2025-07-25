# ADR-001: Stateful Resource Management
- **Status**: Draft
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
## 3. The Provider Contract: Dependency Injection

To establish a link between a `step` and the `resource` it needs, a `step` block will use a `uses` meta-argument. The value of this argument is an HCL expression that references a `resource` instance.

A `runner` manifest declares the *type* of `asset` it is compatible with. The user's `step` block provides the concrete `resource` *instance*.

HCL definition for a runner manifest (e.g., `modules/sql/manifest.hcl`), showing the 'uses "database" {}' block that declares its uses requirement.

```hcl
runner "sql_query" {
  description = "Executes a SQL query."
  # This runner declares it needs a uses that conforms to the "database" asset type.
  uses "database" {}
  
  input "query" {
    type        = string
    description = "The SQL query to execute."
  }
}
```

User's HCL grid file (e.g., `my_test.hcl`), showing a 'resource' block being defined and a 'step' block using the 'uses' meta-argument to reference it.
```hcl
resource "database" "main_db" {
  # Arguments for creating the database resource
  connection_string = "postgres://user:pass@host:port/db"
}

step "sql_query" "get_users" {
  # The step explicitly declares its dependency on the 'main_db' resource.
  # The engine uses this to build the DAG and inject the dependency.
  uses = resource.database.main_db

  arguments {
    query = "SELECT * FROM users;"
  }
}
```

The Go handler for the `step`'s runner would then have a signature that accepts the uses object, ensuring type safety is enforced by the engine at runtime.

Go handler function signature for a step, showing the injected uses interface (e.g., `uses database.Connection`).
```golang
// In the database asset's Go package (e.g., assets/database/):
func init() {
	engine.RegisterAssetInterface("database", reflect.TypeOf((*Connection)(nil)).Elem())
}

// Connection defines the interface a database uses must satisfy.
type Connection interface {
    Exec(ctx context.Context, query string, args ...any) (any, error)
    Close() error
}

// In the sql_query runner's Go package (e.g., modules/sql/):
// OnRunSqlQuery is the handler for the step.
// The `uses` argument is typed to the specific interface it needs.
func OnRunSqlQuery(ctx context.Context, uses database.Connection, input *Input) (any, error) {
    // ... implementation ...
}
```

### The Concurrency Safety Contract

A critical aspect of the uses contract is **concurrency safety**. Because the executor runs steps in parallel, multiple goroutines may attempt to use the same shared `resource` instance simultaneously.

Therefore, any Go object returned by a `resource`'s `Create()` handler **MUST be safe for concurrent use**.

The implementer of an `asset` is responsible for ensuring this thread safety. This typically involves:
* Using mutexes (`sync.RWMutex`) to protect shared state within the object.
* Returning a true connection pool object (like `*sql.DB`) which manages its own internal locking, rather than a single, unsafe connection.
* Designing the object's methods to be re-entrant and free of side effects that could cause race conditions.

Failing to adhere to this contract will lead to data races and unpredictable behavior at runtime. The engine assumes this contract is met and will not provide external locking for `resource` instances.

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

1.  **Unified DAG Construction:** The engine parses all HCL files and builds a single graph containing both `resource` and `step` nodes. A dependency edge is created from a `step` to a `resource` via the `uses` meta-argument, as well as from standard `step.A.output` interpolations.

2.  **Stateful Node Execution:** The executor's worker pool can execute any node whose dependencies are met.
    * When a worker executes a **`resource` node**, it calls the resource's `Create()` method. To ensure this happens exactly once and is concurrency-safe, the executor will use a `sync.Once` primitive for each resource node's creation logic. The stateful object returned by `Create()` is stored in a thread-safe central container, keyed by the resource's unique ID (e.g., `resource.database.main_db`).
    * When a worker executes a **`step` node**, the executor retrieves the required stateful uses object(s) from the central container and injects them as arguments into the `on_run` handler function.

3.  **Resource Lifecycle Management & Cleanup:** The executor manages the resource lifecycle through a robust, two-part strategy to ensure both efficiency and reliability.
    * **Descendant Completion Counter (For Efficiency):** To enable efficient, runtime garbage collection, each `resource` node maintains an atomic counter.
        * *Initialization*: During graph construction, the executor performs a one-time traversal of the `resource`'s entire downstream dependency cone (all `step`s that depend on it, directly or transitively) and initializes the counter to the total number of these `step`s.
        * *Execution*: When a `step` completes (succeeds or fails), it atomically decrements the counter on each of its uses `resource`s.
        * *Trigger*: When a `resource`'s counter reaches zero, the executor immediately schedules its `Destroy()` method.

    * **Guaranteed Cleanup Stack (For Reliability):** To guarantee that resources are always released, the executor maintains an internal LIFO (Last-In, First-Out) stack of cleanup functions.
        * *Registration*: Immediately after a `resource`'s `Create()` method returns successfully, its `Destroy()` function is pushed onto this stack.
        * *Execution*: The main `executor.Run()` function uses a `defer` block that, upon any exit, unwinds this stack and executes each `Destroy()` function. This ensures cleanup even in the case of a panic or early termination.
