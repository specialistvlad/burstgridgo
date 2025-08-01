# ADR-009: Advanced Declarative Engine Features

**Status:** Draft
**Date:** 2025-07-30
**Depends On:** ADR-008

---

## Context and Problem Statement

Following ADR-008, modules are now pure Go, but the engine's capabilities are still basic. It lacks a rich type system for handling complex but common types (e.g., regular expressions). It also cannot enforce declarative validation rules, apply default values, or handle sensitive data based on the manifest definitions, placing this burden on module authors.

---

## Decision Drivers

* **Rich, Validated Core Types:** The engine should support custom types (e.g., `regex`) and declarative validation rules defined in manifests.
* **Declarative Defaults:** Support optional arguments via `default` values defined in the manifest, simplifying module logic.
* **Security:** The engine should be able to handle `sensitive` data and prevent it from being logged.
* **Improved Developer Experience:** Module authors should be able to define their entire contract, including validation, declaratively.

---

## Decision Outcome

We will enhance the `executor` and introduce a **Core Type System**, making the engine aware of the rich semantics defined in module manifests.

1.  **Core Type System:**
    * An internal `TypeRegistry` will be created to map type names from manifests (e.g., `"string"`, `"regex"`) to their corresponding Go types.
    * Core internal types (e.g., `types.Regexp`) will implement standard Go interfaces like `encoding.TextUnmarshaler` to provide automatic validation and conversion.

2.  **Manifest-Aware Executor:** The `executor`'s step-processing pipeline will be expanded to:
    * Apply any `default` values for arguments the user did not provide.
    * Run any declarative `validation` blocks defined in the manifest.
    * Call an optional Go `Validate` function provided by the module.
    * Redact any values from `sensitive` inputs in its logging.

3.  **Final Handler Call:** Only after all checks pass will the `executor` call the module's pure Go handler with the fully processed data.

---

## Validation and Testing

A comprehensive **integration test suite** will be developed to validate each new declarative feature. Specific tests will be created for `default` value application, `validation` block enforcement, custom `type` conversion, and `sensitive` data redaction by running the full application and asserting the expected behavior.

---

## Consequences

### Positive

* **Powerful Declarative Features:** The engine can enforce complex validation and handle custom types based on the manifest alone.
* **Dramatically Simpler Modules:** Removes boilerplate validation and default-handling logic from module code.
* **Improved Security:** The `sensitive` flag provides a built-in, enforced mechanism for redacting secret data.

### Negative

* **Significant Executor Complexity:** The `executor`'s internal pipeline becomes much more complex. Debugging this "magic" will be challenging.
* **Increased Performance Overhead:** The additional validation and processing stages add to the execution time of every step.