# ADR-001: Stateful Resource Management
- **Status**: Implemented
- **Author**: Vladyslav Kazantsev
- **Date**: 2025-07-24

This document outlines the architecture for introducing stateful, managed assets into the workflow engine. This design enables resource sharing (e.g., connection pools, authenticated clients) between steps in a safe, efficient, and declarative manner.

---
## 1. Core Concepts & Naming Convention

The system distinguishes between two fundamental concepts: stateless **Actions** and stateful **Assets**. This introduces the Asset concept to work alongside the existing Action primitives.

| Concept | In Module Manifest (The Blueprint) | In User's HCL (The Instance) |
| :--- | :--- | :--- |
| **The Action** | **`runner`**: Defines a stateless, runnable action. | **`step`**: Executes an instance of a `runner`. |
| **The Asset** | **`asset`**: Defines the schema for a stateful, shareable object. | **`resource`**: Creates a managed instance of an `asset`. |

---
## 2. Lifecycles & Responsibilities

The engine treats `resources` and `steps` as distinct node types within the execution graph. A `resource` **is** a thing, while a `step` **does** a thing.

**Resource Lifecycle Sequence (Text Description):**

1.  The **Executor** schedules a **Resource Node** for creation when it's first needed.
2.  The **Resource Node** executes its `Create()` handler and returns a live object instance, which is stored by the **Executor**.
3.  The **Executor** schedules a dependent **Step Node** for execution.
4.  The **Step Node** receives the shared object instance via its `uses` block dependency.
5.  The **Step Node** completes its `on_run()` logic and notifies the **Executor**.
6.  The **Executor** decrements the resource's descendant counter.
7.  When the counter reaches zero (meaning all direct dependent steps are done), the **Executor** schedules the **Resource Node** for destruction.
8.  The **Resource Node** executes its `Destroy()` handler to release the asset.

#### **`resource` Lifecycle: `Create` & `Destroy`**
A **`resource`** is a **long-lived, stateful asset**. Its lifecycle is managed by two main methods implemented in its Go handler:

* **`Create()`**: Called once by the executor when the `resource` is first needed. Its purpose is to initialize the asset and return the stateful Go object.
* **`Destroy()`**: Called by the executor to release the asset. The current implementation triggers this efficiently via a **direct descendant completion counter** on the `resource` node. A deferred **cleanup stack** in the executor also guarantees `Destroy()` is called when the grid run terminates, regardless of success or failure.

#### **`step` Lifecycle: `on_run`**
A **`step`** is a **transient, stateless action**. Its primary lifecycle method is `on_run()`, which contains the business logic to perform a single, discrete action.

---
## 3. The Provider Contract: Dependency Injection via `uses` block
To establish a safe and explicit link between a `step` and the `resource`(s) it needs, a `step` block uses a dedicated `uses {}` block.

#### HCL Contract
A `runner`'s manifest declares the *types* of assets it is compatible with. The user's `step` block provides the concrete `resource` *instances*.

* **Runner Manifest (`modules/http_request/manifest.hcl`)**
    <br>
    [See an example runner manifest here.](../../modules/http_request/manifest.hcl)

* **User Grid File (`examples/http_request.hcl`)**
    <br>
    [See an example user grid file here.](../../examples/http_request.hcl)

#### Go Handler Contract
The Go handler for the `step` receives its dependencies in a dedicated, type-safe `Deps` struct.

* **Go Handler Implementation (`modules/http_request/module.go`)**
    <br>
    [See an example Go handler here.](../../modules/http_request/module.go)

---
## 4. Asset Type Registration & Validation
To bridge the gap between an HCL asset type (e.g., `"http_client"`) and a Go interface (`*http.Client`), the system maintains a central registry of asset interfaces.

#### **Asset Interface Registry**
Asset authors register their Go interface in their package's `init()` block using the central `registry` package.
<br>
[See an example of interface registration here.](../../modules/http_client/module.go)

#### **Engine Validation Logic**
Before executing a `step`, the executor's `deps_builder` performs validation to verify that the concrete `resource` object's type implements the required interface, preventing runtime panics.