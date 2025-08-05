# ADR-003: Comprehensive System Integration Testing Strategy
- **Status**: Implemented
- **Author**: Vladyslav Kazantsev
- **Date**: 2025-07-27

## 1. Context
The project has several established testing concepts (ADR-002, 003, 004), but lacks a single, unified strategy for high-level testing. We need a robust approach that provides high confidence in the entire application—from CLI parsing to DAG execution—while maintaining a fast feedback loop for local development. This ADR proposes a formal strategy to achieve that.

---
## 2. Decision
We will adopt a **System Integration Testing** model as our primary strategy for ensuring application correctness. This involves testing the application via a refactored, in-process entrypoint with mocked external dependencies. This strategy is defined by the following core mechanics and test plan.

### Core Testing Mechanics
1.  **Testable Entrypoint**: The `main()` function will be refactored into a thin wrapper around a testable `run()` function. This new function will accept context, arguments, and I/O writers to allow for full control within a test environment.
2.  **Test Categorization**: We will use Go build tags (e.g., `//go:build integration`, `//go:build system`) to categorize tests. This enables developers to run fast, relevant subsets locally while ensuring full, comprehensive test execution in the CI pipeline.
3.  **Dynamic Test Scenarios**: All test-specific HCL grid files will be generated dynamically within each test function and written to temporary directories. This ensures tests are fully isolated and self-contained, with no dependency on static example files.

### System Integration Test Plan
The following list represents the comprehensive suite of tests to be implemented using the mechanics described above.

| Category          | Sub-Category     | Feature Under Test                                          |
| :---------------- | :--------------- | :---------------------------------------------------------- |
| **CLI Behavior** | Default Behavior | Displays help text when no grid path is provided.           |
| **CLI Behavior** | Configuration    | Correctly merges HCL configuration from a directory path.   |
| **Core Execution**| Stateless        | Complex data (objects, lists) passes correctly between steps. |
| **Core Execution**| Stateful         | Resource `Create` handler is called only once per instance.   |
| **Core Execution**| Stateful         | All dependent steps receive the exact same resource instance. |
| **Core Execution**| Stateful         | Resource state is correctly modified across multiple steps.   |
| **Core Execution**| Stateful         | Resource `Destroy` handler is called once on cleanup.       |
| **HCL Features** | Dependencies     | Implicit dependency from variable interpolation works.      |
| **HCL Features** | Dependencies     | Explicit `depends_on` correctly forces execution order.     |
| **HCL Features** | Arguments        | Runner receives default value.                              |
| **HCL Features** | Dynamic Blocks   | `count` meta-argument correctly creates multiple nodes.     |
| **HCL Features** | Dynamic Blocks   | `for_each` meta-argument correctly creates dynamic nodes.   |
| **HCL Features** | Dynamic Blocks   | Conditional execution using `count = var.condition ? 1 : 0`.  |
| **DAG / Concurrency** | Graph Shape    | Fan-out execution runs nodes in parallel.                   |
| **DAG / Concurrency** | Graph Shape    | Fan-in synchronization waits for all parallel nodes.        |
| **DAG / Concurrency** | Graph Shape    | Independent parallel tracks execute concurrently.           |
| **Error Handling**| Runtime          | A step times out and correctly fails the run.               |
| **Error Handling**| Runtime          | A resource connection times out during creation.            |
| **Error Handling**| Runtime          | A failing step correctly triggers fail-fast termination.      |
| **Error Handling**| Runtime          | A resource creation failure correctly skips dependents.     |
| **Error Handling**| Load-time        | An invalid HCL file is rejected with a clear error.         |

---
## 3. Consequences

#### Pros 👍
* **High Confidence**: This strategy validates the entire application stack in-process without the brittleness of external services.
* **Fast Feedback Loop**: Developers can run a fast suite of tests locally, improving productivity.
* **Clear Test Plan**: The categorized list of features provides a clear implementation roadmap and ensures comprehensive coverage.
* **Improved Maintainability**: Dynamic, isolated test scenarios are more robust and easier to maintain than tests relying on static files.

#### Cons 👎
* **Upfront Investment**: Requires an initial effort to refactor the `main` function and establish the test harness pattern.
* **CI Configuration**: The CI pipeline must be updated to correctly execute different categories of tests.