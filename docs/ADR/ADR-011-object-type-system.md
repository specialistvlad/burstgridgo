### ADR-011: Object Type System

**Status:** Implementation
**Date:** 2025-08-04
**Depends On:** ADR-010

---

### Context and Problem Statement

With primitives (ADR-009) and collections (ADR-010) in place, the engine's type system is capable but still has a significant gap: it cannot natively represent structured data with named fields.

Many use cases require passing complex data payloads that are not simple lists or key-value maps of a single type. For example, emitting a structured event, configuring a component with multiple heterogeneous parameters, or processing a structured API response. The current workaround involves encoding this data as a JSON string, which is cumbersome for users to write in HCL and completely bypasses our type-safety guarantees.

To enable these advanced use cases and improve developer ergonomics, the type system must be extended to support structured objects.

---

### Decision Drivers

* **Enable Complex Scenarios:** Support for structured objects is a prerequisite for modules that deal with event payloads, complex configurations, or rich API interactions.
* **Improve Manifest Expressiveness:** Allowing module authors to define `object({ name = string, timeout = number })` makes the contract of the module explicit, self-documenting, and easier to understand.
* **Provide Structural Type Safety:** The engine should be able to validate not just the type of an input, but its shape—ensuring required attributes are present and that their values conform to the specified types.
* **Enhance User Experience:** Users should be able to define structured data using natural HCL syntax rather than embedding and escaping JSON strings.

---

### Decision Outcome

We will enhance the type system to support `object` types, allowing for both loosely-defined and strictly-defined structures.

1.  **Scope:** This ADR covers two forms of object types:
    * **Generic Objects:** Using `type = object({})` will define an input that accepts any object and decodes it into a `map[string]any` in Go.
    * **Structurally-Typed Objects:** Using `type = object({ key = type, ... })` will define an input that enforces a specific structure and can be decoded into a matching Go `struct`.
    * Due to the recursive nature of the parser, this scope implicitly includes support for **nested objects** (e.g., `object({ a = object({ ... }) })`) and attributes of type **`any`**.

2.  **Tagging Convention for Module Authors:** A two-level tagging system will be used to decode objects into Go structs:
    * The **`bggo:"..."`** tag will be used exclusively on the fields of a module's top-level `Input` struct to map HCL arguments.
    * The **`cty:"..."`** tag (from the `go-cty` library) will be used on the fields of any *nested* Go struct that represents an HCL object's structure.
    * *Example:*
        ```go
        // Inner struct uses 'cty' tags for its fields.
        type Payload struct {
            TransactionID string `cty:"transaction_id"`
        }
        // Top-level input struct uses 'bggo' tags for its fields.
        type ModuleInput struct {
            EventData Payload `bggo:"event_data"`
        }
        ```

3.  **HCL Type Parser (`hcl/translate_type.go`):** The `typeExprToCtyType` function will be enhanced to parse the `object` function call syntax. It will recursively parse the keys and type expressions within the object constructor to build the complete `cty.Object` type.

4.  **Startup Type Parity Check (`registry/validate.go`):** The registry validation logic will be extended to compare manifest object definitions with their Go implementations, checking for compatible structures and types, and respecting the new tagging convention.

5.  **Runtime Validation:** No significant changes are anticipated for the `hcl.Converter`. The `go-cty` library's existing conversion logic will handle decoding and validation, enforcing the structure defined by the manifest's `cty.Type`.

---

### Validation and Testing

A new integration test file (`integration_tests/type_system/object_type_validation_test.go`) will be created to fully validate the implementation.

* **Startup Validation Test:** Assert that a mismatch between a manifest's `object` definition and the implementing Go struct causes the application to fail on startup.
* **Runtime "Happy Path" Tests:**
    * A test for a generic `object({})` input, asserting it correctly decodes into a `map[string]any`.
    * A test for a structurally-typed object, asserting it correctly decodes into a specific Go struct.
    * A test for a **nested object** to validate multi-level decoding.
    * A test for an object containing an attribute of type **`any`** to validate flexible payloads.
* **Runtime Failure Test:** Assert that providing an HCL object where an attribute's value violates its defined type causes the run to fail with a clear error message.

---

### Consequences

#### Positive

* Unlocks the ability to pass complex, structured data, a major feature for real-world usability.
* Vastly improves the readability and self-documenting nature of module manifests.
* Introduces structural type safety, a higher level of validation than just primitives or collections.
* Completes the core of our planned type system, making the platform significantly more powerful and robust.

#### Negative

* Adds non-trivial complexity to the `hcl/translate_type.go` parser. This is a necessary and justified trade-off for the immense gain in functionality.
* Requires module authors to understand the distinction between `bggo` and `cty` tags. This must be clearly documented.