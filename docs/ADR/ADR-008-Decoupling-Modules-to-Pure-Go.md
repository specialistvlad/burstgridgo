# ADR-008: Decoupling Modules to Pure Go

**Status:** Implemented
**Date:** 2025-07-30
**Depends On:** ADR-007

---

## Context and Problem Statement

Following ADR-007, the source of configuration is abstracted. However, the modules themselves remain tightly coupled to the HCL ecosystem. Their `Input` structs require `hcl` tags, and their handlers must return `cty.Value`. This prevents modules from being pure, portable Go code and complicates testing.

---

## Decision Drivers

* **Module Purity:** Module authors should be able to write pure Go logic without knowledge of HCL or CTY.
* **Simplified Testing:** Unit testing a module's business logic should not require constructing complex `cty.Value` objects.
* **True Agnosticism:** To realize the benefit of a pluggable loader, the code that consumes the configuration must also be format-agnostic.

---

## Decision Outcome

We will refactor modules to be "pure Go" and update the `executor` to act as the translation bridge using the `Converter` interface from ADR-007.

1.  **Format-Agnostic Module Contract:**
    * **Input Contract:** All module inputs **must** be defined as a pure Go `struct`. This struct serves as the container for all arguments a module receives, ensuring a consistent and predictable shape. To map configuration keys (e.g., `snake_case`) to `PascalCase` Go fields, this `Input` struct must use a generic `bggo:"..."` tag.

    * **Output Contract:** Similarly, all module outputs **must** be returned as a pure Go `struct`. Simple values, like a single string or integer, must be wrapped within a field of an output struct. To ensure correct attribute naming in the engine, this `Output` struct must use `cty:"..."` tags on its fields. We use the standard `cty` tag here as a pragmatic choice to leverage the robust, third-party `gocty` library for this translation, avoiding the need to maintain a custom implementation for a solved problem.
2.  **Registry Update:** The `registry` will be updated to store the `reflect.Type` of each module's `Input` and `Deps` structs.

3.  **Executor as Translation Bridge:** The `executor` will be refactored to:
    * Receive a `Converter` interface during initialization.
    * Before running a step, use reflection to create an instance of the module's plain Go `Input` struct.
    * Call the `converter.DecodeBody(...)` method, whose implementation uses reflection to read the `bggo` tags and perform the just-in-time data translation.
    * After the pure Go handler runs, call `converter.ToCtyValue(...)` to translate the native Go result back into the engine's internal value system, respecting `cty` tags on any returned structs.

---

## Consequences

### Positive

* **Achieves Agnostic Modules:** Module code becomes pure, simple, and highly testable Go.
* **The "Contract" is Explicit:** A module's contract is its Go function signature combined with the declarative `bggo` (for inputs) and `cty` (for output structs) tags on its data structures.
* **Simplifies Module Authoring:** Lowers the barrier to entry for creating new modules.

### Negative

* **Executor Complexity:** The `executor` becomes a more complex component, now heavily reliant on reflection.
* **Performance Overhead:** Reflection and two-way type conversions for every step will introduce a performance cost.
* **Minor Abstraction Leak:** Module authors must be aware of the `cty` tag for output structs, which is a minor dependency on the engine's internal type system.