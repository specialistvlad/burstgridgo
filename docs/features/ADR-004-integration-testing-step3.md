# ADR-004: Expanding Integration Test Coverage
- **Status**: Draft
- **Author**: Vladyslav Kazantsev
- **Date**: 2025-07-24

---
## 1. Context

The initial integration test for stateless workflows (`ADR-002`) has been successfully implemented. It validates the core "happy path" of parsing an HCL file, building a simple dependency graph, and passing data between two steps.

However, a review of our test suite reveals significant gaps. We currently have no coverage for dynamic HCL features (`count`, `for_each`), complex graph topologies (fan-in/fan-out), or critical failure scenarios. To ensure the engine is robust and to prevent future regressions, we must systematically expand our integration test suite to cover these real-world use cases.

---
## 2. Decision

We will implement a new suite of integration tests, using the established mock injection pattern, to validate the engine's behavior across a wide range of configurations and scenarios. The following test cases represent the minimum required coverage to be added.

#### HCL Syntax & Dynamic Features
* **`count` Meta-Argument**: A test where a `step` block with `count = 3` correctly creates three distinct nodes in the execution graph.
* **`for_each` Meta-Argument**: A test where a `step` block with `for_each` over a map or set creates a unique node for each item.
* **Conditional Step Execution**: A test using `count = var.condition ? 1 : 0` to verify that a step is correctly included or excluded from the graph based on a boolean variable.
* **Explicit & Multiple Dependencies**: A test where a step with `depends_on = ["step.A", "step.B"]` only executes after both dependencies have successfully completed.
* **Complex Data Marshalling**: A test that passes a complex data structure (e.g., a nested object containing a list) from one step to another and verifies its integrity.

#### Graph Topologies & Execution Flow
* **Fan-Out Execution**: A test where one initial step is a dependency for several other steps, verifying the dependent steps run in parallel.
* **Fan-In Execution**: A test where a final step depends on several parallel steps, verifying it only runs after all of them have completed.
* **Independent Parallel Tracks**: A test with two or more completely independent dependency chains in the same grid, verifying they execute concurrently without interference.

#### Failure & Edge Case Handling
* **Step Failure Propagation**: A test where a mock handler returns an error, verifying that its dependent steps are skipped and the executor reports the failure correctly.
* **Input Default Value Application**: A test that calls a runner without providing an optional argument, verifying that the handler receives the correct default value specified in the runner's manifest.
* **Step Timeout Enforcement**: A test where a mock handler simulates a long-running process, verifying that the step is terminated with a timeout error after its configured duration.

#### Configuration & File Loading
* **Multi-File Directory Loading**: A test that loads a directory containing multiple `.hcl` files, verifying that the configurations are correctly merged into a single, cohesive execution graph.

---
## 3. Consequences

#### Pros
* **Increased Confidence**: Provides high confidence that the engine correctly handles complex, real-world workflows, not just simple linear cases.
* **Regression Safety Net**: Creates a robust safety net that will catch regressions in core logic as we refactor or add new features.
* **Improved Reliability**: Explicitly testing failure modes (errors, timeouts) will lead to a more predictable and resilient application for end-users.
* **Actionable Backlog**: This document provides a clear, categorized backlog of testing work for the development team.

#### Cons
* **Development Time**: Implementing this comprehensive suite requires a significant time investment.
* **Increased CI/CD Duration**: A larger test suite will increase the runtime of our continuous integration pipeline.
* **Maintenance Overhead**: These tests will need to be maintained and updated as the engine's architecture evolves.