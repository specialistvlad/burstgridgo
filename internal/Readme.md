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

* **2. Module Author Defines:**
    * **Manifest File** (`.hcl`): Defines the public API of a 'runner' or 'asset'.
    * **Go `Module`** (`.go`): Implements the component's business logic in pure Go.
        * **Inputs:** The module's `Input` **must** be a pure Go `struct` that uses a generic `bggo:"..."` tag to map configuration keys to its fields.
        * **Outputs:** If a handler returns data, it **must** be a pure Go `struct` that uses `cty:"..."` tags on its fields to map them to the attribute names expected by the engine.

* **3. User Defines:**
    * **User Grid File** (`.hcl`): Creates instances ('steps' and 'resources') of the components defined by the module author to describe a workflow.

* **4. Engine on Run (Core Packages):**
    * **`config`**: Defines the **format-agnostic** configuration `Model` and the core `Loader` and `Converter` interfaces. This is the pure data model the rest of the engine works with.
    * **`hcl`**: Provides the concrete **HCL implementation** of the `config.Loader` and `config.Converter` interfaces. It's responsible for all file parsing and HCL-specific data binding, encapsulating all HCL parsing structs internally.
    * **`app`**: The main application orchestrator. It initializes a `config.Loader` (e.g., the `hcl.Loader`), drives the loading process, and manages the overall execution flow.
    * **`dag`**: The graph-building layer. It takes the **format-agnostic `config.Model`** and builds a validated execution graph. **It does not run the graph.**
    * **`executor`**: The execution layer. It takes a pre-built graph and manages its concurrent execution. It uses the `config.Converter` to perform just-in-time data binding by reading the `bggo` tags from the module's Go struct via reflection.

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

2.  **Module Registration Integrity**:
    * This is the primary static validation step.
    * The engine ensures that every lifecycle handler named in a manifest (e.g., `lifecycle { on_run = "OnRunMyModule" }`) corresponds to a Go handler that was actually registered in the `registry`.
    * This check prevents fatal runtime errors due to typos or mismatches between the configuration and the compiled Go code. The flexible `bggo` tag now manages the field-level mapping, which is handled at runtime.

### Phase 2: Per-Step Runtime Pipeline

This pipeline is executed by the `executor` for **every `step` node** in the directed acyclic graph (DAG) during a run. It describes the "just-in-time" process of preparing data for, validating, and executing a module's business logic.

The pipeline ensures that a module handler receives a pure, simple Go struct containing data that has been thoroughly vetted.

The sequence is as follows:

1.  **Expression Evaluation**:
    * The `executor` first identifies any HCL expressions within the step's argument block (e.g., `args = { message = "Hello, ${step.A.output.name}!" }`).
    * It resolves these expressions using the current evaluation context, substituting them with their real, calculated values.

2.  **Input Translation (`ADR-008`)**:
    * The `executor` creates a new, zero-value instance of the module's pure Go `Input` struct.
    * It uses the `config.Converter` interface to decode the step's argument data. The converter uses Go's reflection to inspect the `Input` struct's fields, reads the `bggo:"..."` tags to determine the mapping key, and then populates the fields with the configuration data.

3.  **Declarative Manifest Validation (`ADR-009 - Future Work`)**:
    * The `executor` will inspect the runner's manifest for any declarative `validation {}` blocks associated with the inputs.
    * It will enforce these rules against the data now present in the populated Go `Input` struct.
    * If any of these validations fail, the step fails, and the execution of the graph is halted.

4.  **Imperative Handler Validation (`ADR-009 - Future Work`)**:
    * After passing the manifest's declarative checks, the engine will check if the module's Go handler struct implements an optional `Validate() error` method.
    * If it exists, the `executor` calls this method. This allows the module author to perform complex, cross-field business logic validation.
    * If this method returns an error, the step fails.

5.  **Execution**:
    * Only after all preceding stages does the `executor` finally call the module's `OnRun(ctx, input)` method.
    * The handler receives the fully populated, translated, and validated pure Go `Input` struct, allowing the author to focus entirely on the module's business logic.

6.  **Output Translation (`ADR-008`)**:
    * After the pure Go handler executes and returns its `Output` struct, the `executor` calls the `converter` again.
    * The converter translates this native struct back into the engine's internal representation (e.g., a `cty.Value` object). It inspects the `cty:"..."` tags on the struct's fields to ensure the output can be correctly used by downstream steps that depend on it.