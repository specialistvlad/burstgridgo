# ADR-007: Unified Configuration Loading

**Status:** Implemented
**Date:** 2025-07-30

## Context and Problem Statement

The application architecture tightly couples configuration loading with application orchestration. It enforces a rigid distinction between "module paths" and "grid paths," which prevents a holistic view of the system's configuration and makes it impossible to support multiple configuration formats or a unified file schema. To improve separation of concerns and enable future flexibility, we must abstract the configuration loading process.

## Decision Drivers

* **Separation of Concerns:** The core application should orchestrate workflows, not parse files.
* **Flexibility:** Pave the way for supporting multiple configuration formats and a unified file schema where definitions and instances can coexist.
* **Unified Configuration View:** Treat all configuration files as a single collection, loading them into one holistic, in-memory model.

## Decision Outcome

We will establish a clean boundary between configuration interfaces (in `internal/config`) and their concrete implementations (e.g., `internal/hcl`). The `app` package will act as a **path orchestrator**.

1.  **Interface Definition (`config` package):**
    * **`Model`**: A standard struct holding collections of all Definitions (`RunnerDefinition`, `AssetDefinition`) and Instances (`Step`, `Resource`).
    * **`Converter`**: An interface responsible for format-specific data binding.
    * **`Loader`**: An interface whose `Load()` method accepts a **slice of file paths** and returns a `*Model` object and a `Converter`.

2.  **Path Orchestration (`app` package):**
    * The `app` package will be responsible for collecting configuration paths from all user-provided sources (e.g., `--grid` and `--modules-path` flags).
    * It will **merge these paths into a single, unified slice** before passing them to the `Load()` method of a `Loader` implementation.

3.  **Initial Implementation (`hcl` package):**
    * The first loader will be an `hcl.Loader`.
    * It will accept a single collection of paths from the `app` layer. It will recursively discover all `.hcl` files within these paths.
    * It will **inspect the content of each file** to dynamically identify and parse any top-level block (`runner`, `asset`, `step`, `resource`), treating all `.hcl` files as a homogeneous collection. All parsed objects will be merged into the unified `Model` struct.

## Consequences

### Positive

* **Establishes a Clean Boundary:** The application core is no longer responsible for file parsing.
* **Creates a Unified Model:** The rest of the application consumes a single, complete configuration object.
* **Foundation for Polyglot Support:** Creates the necessary abstraction to support multiple configuration languages.
* **Flexible File Structure:** Allows users to co-locate component definitions and their instances in the same file if desired.

### Negative

* **Introduces New Interfaces:** Adds new interfaces to the codebase.
* **Adds Loader Complexity:** The `hcl.Loader` becomes more complex, as it must now dynamically inspect file content to determine how to parse each block rather than relying on file paths.