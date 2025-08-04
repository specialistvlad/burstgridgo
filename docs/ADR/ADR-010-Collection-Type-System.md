### ADR-010: Collection Type System

**Status:** Implemented
**Date:** 2025-08-04
**Depends On:** ADR-009

---

### Context and Problem Statement

Following the successful implementation of ADR-009, the engine now robustly enforces primitive types (`string`, `number`, `bool`) as defined in module manifests. This provides a solid foundation for type safety.

However, the system cannot yet handle collection types. Many critical load testing and automation scenarios rely on processing lists of items (e.g., URLs to hit, user credentials to test), sets of unique identifiers, or maps of configuration data. Without native support for these collections, module authors are forced to use less safe workarounds, such as encoding data into a single string.

To make `burstgridgo` a truly powerful and practical tool, we must extend the type system to support these fundamental collection structures.

---

### Decision Drivers

* **Unlock Critical Use Cases:** Supporting collections is essential for iterating over datasets, a core requirement for most non-trivial load tests.
* **Improve Module Authoring:** Providing explicit `list(T)`, `map(T)`, and `set(T)` types makes module manifests far more expressive, readable, and self-documenting.
* **Maintain and Extend Type Safety:** The type-safety guarantees established in ADR-009 must be extended to collections, preventing errors like a `list(string)` containing a `number`.
* **Build on Existing Foundation:** This is the natural, incremental next step, extending the parsing and validation infrastructure we have already built.

---

### Decision Outcome

We will enhance the type system to support `list`, `map`, and `set` collections containing primitive types.

1.  **Scope:** The scope of this ADR is to support `list(T)`, `map(T)`, and `set(T)`, where `T` **must be a primitive type** (`string`, `number`, or `bool`). Nested collections (e.g., `list(list(string))`) and complex object types are out of scope and will be addressed in a subsequent ADR.

2.  **HCL Type Parser (`hcl/translate.go`):** The `typeExprToCtyType` function will be significantly enhanced. It will be updated to parse HCL's function-call syntax (e.g., `list(string)`).
    * It will identify `list`, `map`, and `set` as known type constructor functions.
    * It will validate that they are called with exactly one argument.
    * It will parse the argument to determine the element type (`T`).
    * It will construct and return the appropriate `cty` collection type (e.g., `cty.List(cty.String)`).

3.  **Startup Type Parity Check (`registry/validate.go`):** The `ValidateRegistry` function will be updated to correctly compare collection types between the manifest and the Go implementation.
    * For a manifest's `list(string)`, it will check for compatibility with a Go `[]string`.
    * For a `map(number)`, it will check for compatibility with a Go `map[string]int` (or other numeric types).
    * For a `set(bool)`, it will check for compatibility with a Go `[]bool` (as sets are represented as slices in this context).
    * Any incompatibility will result in a clear startup failure.

4.  **Runtime Validation:** No changes are expected in the `hcl.Converter`. The existing logic from ADR-009, which uses `convert.Convert(value, manifestType)`, will automatically handle collections. The `go-cty` library is capable of converting an HCL tuple to a typed list and will return an error if any element in the collection violates the element type constraint, giving us runtime validation for free.

---

### Validation and Testing

A new integration test file (`integration_tests/type_system/collection_type_validation_test.go`) will be created to validate the implementation. It will include:

* **Startup Validation Test:** An attempt to load a module where the manifest declares `list(string)` but the Go struct field is an incompatible `[]int` will be asserted to fail on startup.
* **Runtime Validation Tests:**
    * A test passing a valid `list(string)` to a module and asserting success.
    * A test passing a valid `map(number)` to a module and asserting success.
    * A test passing an invalid list containing a `number` to an input expecting `list(string)` and asserting that the run fails with a clear element type-mismatch error.

---

### Consequences

#### Positive

* Unlocks a vast range of common and essential use cases.
* Greatly improves the expressiveness and clarity of module manifests.
* Extends the engine's type safety guarantees to collections, catching more user errors early.

#### Negative

* Adds necessary complexity to the HCL type parser in `hcl/translate.go`. This is a reasonable trade-off for the functionality gained.