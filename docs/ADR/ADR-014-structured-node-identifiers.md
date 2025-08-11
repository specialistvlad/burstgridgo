# ADR-014: Structured Node Identifiers

- **Status**: Implemented
- **Date**: 2025-08-10
- **Authors**: Vladyslav Kazantsev

---

## Context and Problem Statement

The system currently represents the identity of all graph nodes (steps and resources) using raw strings. These string IDs are constructed in the `builder` package using `fmt.Sprintf` and are deconstructed in multiple places across the `builder` and `executor` packages using regular expressions and manual string manipulation. Moreover, the logic for parsing these strings is duplicated across packages.

This approach has several significant drawbacks:

1.  **Brittleness**: The logic for creating and parsing these IDs is scattered. A change to the ID format (e.g., adding a new component) requires a developer to find and update every instance of string formatting and parsing, which is error-prone.
2.  **Lack of a Single Source of Truth**: There is no single, authoritative "schema" for a node ID. Different parts of the system make assumptions about the format, violating the "Don't Repeat Yourself" (DRY) principle and leading to potential inconsistencies.
3.  **No Compile-Time Safety**: A typo in a `fmt.Sprintf` format string or a bug in a regular expression can only be caught at runtime, making the system harder to debug and maintain.
4.  **Code Obfuscation**: Complex parsing logic, especially in performance-sensitive areas like the `executor`'s evaluation context builder, makes the code harder to read and reason about.

---

## Decision Drivers

- The need to improve long-term maintainability and reduce the risk of introducing bugs during future refactoring.
- The desire for a single, authoritative source of truth for node identity.
- The goal of leveraging Go's type system to provide compile-time safety and improve code clarity.
- The architectural principle of high cohesion, ensuring that related logic is grouped together.

---

## Considered Options

1.  **Status Quo**: Continue using string-based identifiers, potentially adding more comments or constants to mitigate the risks. This was rejected as it does not solve the underlying architectural problem.
2.  **Centralized String Utilities**: Create a package with utility functions for creating and parsing ID strings, but continue passing raw strings around the application. This is an improvement but lacks the full benefit of a dedicated type.
3.  **Dedicated `nodeid` Package with a Structured Type**: Create a new, self-contained package (`internal/nodeid`) that defines a structured `Address` type. This type would encapsulate all logic for formatting, parsing, validation, and manipulation.

---

## Decision Outcome

We will proceed with **Option 3: Dedicated `nodeid` Package with a Structured Type**. This approach provides the most robust and maintainable solution by creating a single source of truth for node identifiers and leveraging Go's type system.

## Implementation Strategy

The refactoring will be executed in a phased approach to minimize disruption and ensure the integration test suite remains stable at each stage.

1.  **Encapsulate the ID Field**: The `node.Node.id` field will be made unexported. A public `ID() string` getter will be introduced and used throughout the codebase. This initial, purely mechanical change ensures all access goes through a single method without altering the underlying type, allowing the test suite to pass.
2.  **Introduce the Structured Type**: The underlying `node.Node.id` field will be changed from `string` to `*nodeid.Address`. The `ID()` getter will be updated to return `n.id.String()`, maintaining the external contract. The `builder` package will be updated to construct these new `nodeid.Address` objects, and the `executor` will be refactored to consume them, removing fragile string parsing.

---

## Consequences

### Positive

-   **Increased Type Safety**: The compiler will now enforce the correct usage of node identifiers, preventing a class of runtime errors.
-   **Improved Maintainability**: The logic for handling node IDs is now centralized in the `nodeid` package. Future changes to the ID format only need to happen in one place.
-   **Enhanced Code Clarity**: The `executor` and `builder` packages are simplified by removing manual string parsing and regex matching, making the code easier to read and understand.
-   **Single Source of Truth**: The `nodeid` package and its `Address` struct become the definitive source of truth for the structure and format of a node identifier.

### Negative

-   A new internal package dependency (`internal/nodeid`) is introduced.
-   Developers new to the codebase will need to familiarize themselves with the `nodeid.Address` struct, though its benefits are expected to outweigh this minor learning curve.