# ADR-013: Extract Generic DAG Logic into a Separate Package

**Status:** Implemented

**Context:**

The `internal/builder` package is currently responsible for two distinct concerns:
1.  Translating application-specific configuration (`Steps`, `Resources`) into a set of nodes.
2.  Managing the graph topology itself: storing nodes, adding edges between them, and performing graph-wide operations like cycle detection.

This mixing of responsibilities makes the `builder` package more complex than necessary. The core graph logic is tightly coupled to the application's domain objects, making it difficult to reuse and harder to test in isolation. The `builder.Node` struct contains both static configuration data and fields for dynamic execution state, further violating the Single Responsibility Principle.

**Decision:**

We will create a new, generic package: `internal/dag`.

1.  **`dag` Package Responsibilities:** ✅ **Done.** This package will be solely responsible for representing a Directed Acyclic Graph. It will provide a simple API to manage the graph's topology. It will have no knowledge of the application's business domain (steps, resources, HCL, etc.).
    -   ✅ **Done.** It will define a `dag.Graph` and a minimal `dag.Node`.
    -   ✅ **Done.** It will provide methods like `AddNode(id string)`, `AddEdge(fromID, toID string)`, and `DetectCycles()`.

2.  **Refactor `builder`:** ✅ **Done.** The `builder` package will become a client of the new `dag` package.
    -   ✅ **Done.** `builder.Graph` will hold an instance of `dag.Graph`.
    -   ✅ **Done.** The `builder` will use the `dag` package to perform all topology manipulations. Its own responsibility will be reduced to translating the application config into calls to the `dag` package.

3.  **Testing Requirement:** ✅ **Done.** The new `dag` package must be developed with simplicity as a core principle and must have 100% unit test coverage.

**Implementation Notes:**

* The `internal/dag` package was successfully created with robust, thread-safe methods and 100% unit test coverage.
* The `internal/builder` package was successfully refactored to delegate all graph operations to the `dag` package. Existing integration tests pass, confirming the refactor did not introduce regressions.
* **Technical Debt:** To maintain backward compatibility with downstream consumers (e.g., the `executor` package) and avoid breaking changes, the legacy `Deps` and `Dependents` map fields on `builder.Node` were intentionally kept. These are populated as a temporary bridge to allow methods like `SetInitialCounters` to function without modification.
* **Next Steps:** A follow-up plan to remove this technical debt by refactoring the consumers and then deleting the legacy fields has been documented in **ADR-014**.

**Consequences:**

**Positive:**
-   **Improved Separation of Concerns:** The logic for graph theory is cleanly separated from the application's business logic.
-   **Increased Reusability:** The `dag` package will be a generic, self-contained component that could be used elsewhere.
-   **Enhanced Testability:** The `dag` package can be tested exhaustively in isolation, confirming its correctness. The `builder` package's tests can then focus purely on its translation logic.
-   **Improved Code Clarity:** Both packages become simpler and easier to reason about due to their focused responsibilities.

**Negative:**
-   **Increased Package Count:** Introduces one new package to the project, slightly increasing the overall structural complexity. This is a minor and acceptable trade-off for the benefits gained.