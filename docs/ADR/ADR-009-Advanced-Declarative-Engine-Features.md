# ADR-009: Primitive Type System

**Status:** Implemented
**Date:** 2025-08-04
**Depends On:** ADR-008

---

## Context and Problem Statement

Following ADR-008, modules now have a "pure Go" contract. However, the engine does not yet enforce the data types declared in a module's manifest. For example, if a manifest declares an input `type = number`, the engine will still attempt to coerce that value into a `string` if the backing Go struct field is a string. This type coercion is driven by the Go implementation, not the manifest's explicit contract.

This creates a disconnect where the manifest is not the single source of truth. To build a robust and reliable system, the engine must be able to validate that user-provided configuration values adhere to the type contract defined in the manifest.

While a basic mechanism for handling `default` values currently exists within the HCL-specific translation layer, a formal type system is the necessary foundation to make defaults, validation, and other declarative features truly format-agnostic.

This ADR addresses the first, most fundamental step: implementing a primitive type system.

---

## Decision Drivers

* **Module Contract Integrity:** The manifest must be the single source of truth. The engine must enforce the types it declares.
* **Improved User Experience:** Users should receive clear, early, and actionable error messages if they provide a value of the wrong type (e.g., `"hello"` for a `number` input).
* **Foundation for Future Features:** A strong type system is the prerequisite for `ADR-010` (Collection Types), `ADR-011` (Structural Types), and future declarative validation features.
* **Incremental Delivery:** We have explicitly chosen to implement only primitive types first to reduce risk, manage complexity, and deliver a foundational improvement quickly.

---

## Decision Outcome

We will enhance the configuration loader and converter to formally support and enforce primitive types, and we will add a strict type parity check at startup.

1.  **Scope:** The scope of this ADR is limited to the following types declared in manifests:
    * `string`
    * `number`
    * `bool`
    * `any` (will continue to be supported as an escape hatch)

2.  **Startup Type Parity Check (`registry/validate.go`):** The `ValidateRegistry` function will be enhanced to perform a strict type compatibility check at application startup. For each module input:
    * It will parse the `cty.Type` from the manifest's `type` attribute.
    * It will find the corresponding Go struct field via its `bggo` tag and infer that field's native `cty.Type`.
    * **Rule:** If the manifest specifies a primitive type (`string`, `number`, `bool`), the inferred Go type must be strictly compatible. If not, the application will **fail to start** with a clear error detailing the mismatch.
    * **Rule:** If the manifest specifies `type = any`, the engine will permit any compatible Go type for the tagged field but will **log a warning**, encouraging the author to use a more specific type.

3.  **HCL Loader (`hcl/translate.go`):** The HCL translation logic will be updated. Instead of using a placeholder, it will now properly evaluate a manifest's `type` expression (e.g., `string`) and store the resulting `cty.Type` object in the format-agnostic `config.InputDefinition` model.

4.  **HCL Converter (`hcl/converter.go`):** The `DecodeBody` method will be enhanced. Before decoding a value into a Go struct, it will first use the `cty.Type` from the manifest's definition to perform a strict type check and conversion on the user-provided value. If the conversion fails, the process will stop and return an error.

---

## Validation and Testing

* A new integration test (`TestErrorHandling_TypeMismatch_FailsRun`) will be created to prove that providing a value that does not match the manifest's declared primitive type results in a clear error during the run.
* A new startup validation test (`TestStartupValidation_ManifestGoTypeMismatch_Fails`) will be created. It will attempt to load a module where the manifest (`type = number`) and the Go struct (`string`) are incompatible and assert that the application fails to start.
* Existing integration tests will be updated to use explicit primitive types to ensure they are compatible with the new, stricter checking.

---

## Consequences

### Positive

* The engine will now enforce the primitive type contract defined in module manifests.
* Strict type parity between a module's manifest and its Go implementation is now guaranteed at startup.
* Users will get clearer and earlier feedback on configuration errors.
* Establishes a solid, tested foundation for subsequent type system enhancements (ADR-010, ADR-011).

### Negative

* Adds a small amount of logical complexity to the configuration loading and validation pipeline.