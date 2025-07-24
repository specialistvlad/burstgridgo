# Architecture

This document provides a deep look into the internal architecture of `burstgridgo`. Understanding these concepts is essential for contributing new features or runners.

The architecture is designed to be **declarative and extensible**. It cleanly separates the **definition** of a task (a `runner`'s public API) from its **implementation** (a Go `handler` function) and its **execution** (a `step` block in a user's grid).

---

## The Core Components

There are three key concepts you must understand: the **Runner Definition**, the **Go Handler**, and the **Step**.

### 1. The Runner Definition (`manifest.hcl`)

A Runner Definition is a `.hcl` file that defines the public API or *schema* for a runner. It specifies what inputs it accepts, what outputs it produces, and which Go function to call. These files live inside each module's directory (e.g., `modules/http_request/manifest.hcl`).

* **`runner "type" {}`**: The main block defining the runner's type.
* **`input "name" {}`**: Defines an input argument, its type, and whether it's optional.
* **`output "name" {}`**: Defines a value that the runner will produce.
* **`lifecycle {}`**: Maps an execution event (like `on_run`) to the registered name of a Go handler.

[Placeholder: Example of a runner "http_request" definition from a manifest.hcl file, showing input, output, and lifecycle blocks.]

### 2. The Go Handler (`module.go`)

This is the Go code that implements the runner's logic. It's a standard Go function that takes a context and a pointer to an `Input` struct, and returns a `cty.Value` and an `error`.

* **`Input` struct**: A Go struct that maps to the `input` blocks from the manifest. HCL tags (`hcl:"..."`) are used for decoding.
* **Handler function**: The function signature is `func(ctx context.Context, input *Input) (any, error)`. The `any` return value must be a `cty.Value` or `nil`.
* **Registration**: The handler is registered with the engine in an `init()` block, associating its string name (from the `lifecycle` block) with the function itself.

[Placeholder: Example of a Go handler implementation from a module.go file, showing the Input struct, the OnRunHttpRequest function, and the init() registration block.]

### 3. The Step (User Grid File)

A `step` is an **instance** of a runner that a user defines in their grid file (e.g., `my_test.hcl`). It tells the engine to execute a specific runner with specific arguments.

* **`step "type" "name" {}`**: The block that creates an execution step. The `"type"` must match a defined `runner` type, and the `"name"` is a unique identifier for this instance.
* **`arguments {}`**: This block contains the actual values for the `input`s defined in the runner's manifest.
* **`depends_on = [...]`**: Explicitly defines dependencies on other steps.

[Placeholder: Example of a user's grid file showing a step "http_request" and a dependent step "print" that uses its output.]

---

## Execution Flow

Here is how the components work together when the application runs:

1.  **Startup: Discovery & Registration**
    * The engine calls `engine.DiscoverRunners()` to scan the `modules/` directory. It parses all `manifest.hcl` files and populates a `DefinitionRegistry` map, keyed by runner type (e.g., `"http_request"`).
    * At the same time, Go's runtime executes the `init()` function in each `module.go` file. These functions call `engine.RegisterHandler()`, populating a `HandlerRegistry` map that links handler names (e.g., `"OnRunHttpRequest"`) to the actual Go functions.

2.  **Grid Parsing & DAG Construction**
    * The engine parses the user's grid file(s) into a list of `*engine.Step` structs.
    * The `dag.NewGraph()` function processes this list, building a dependency graph. It creates nodes for each step and adds edges for both explicit (`depends_on`) and implicit (`step.A.output.B`) dependencies. A cycle check is performed.

3.  **Step Execution (The Executor's Role)**
    * The DAG executor starts a pool of workers and feeds them nodes that have no pending dependencies.
    * When a worker executes a step (e.g., `step "http_request" "get_homepage"`):
        1.  It looks up the `RunnerDefinition` for `"http_request"` in the `DefinitionRegistry`.
        2.  It finds the handler name `"OnRunHttpRequest"` from the definition's `lifecycle` block.
        3.  It looks up the `OnRunHttpRequest` function in the `HandlerRegistry`.
        4.  It builds an HCL evaluation context. For any dependencies of this step, it takes their raw output (e.g., a `cty.Value`) and makes it available under the `step.<dep_name>.output` variable.
        5.  It decodes the `arguments` block from the HCL into the Go `Input` struct.
        6.  It calls the Go handler function.
        7.  The handler returns a direct `cty.Value` (e.g., `{"status_code": 200, "body": "..."}`).
        8.  The executor stores this `cty.Value` as the output of the "get_homepage" node, marking it as complete. This may unblock other steps that depend on it.

This process repeats until all nodes in the DAG are complete.