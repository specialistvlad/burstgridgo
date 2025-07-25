# ADR-003: Integration Testing for Stateful Resources
- **Status**: Proposed
- **Author**: Vladyslav Kazantsev
- **Date**: 2025-07-24

---
## 1. Context

Following the implementation of `ADR-002`, we have established a testing strategy for stateless workflows. However, a critical piece of the engine's architecture, the management of stateful **assets** and **resources** as defined in `ADR-001`, remains completely untested.

This includes the resource lifecycle (`Create`, `Destroy`), the correct injection of shared resource instances into steps via the `uses` block, and the guaranteed cleanup of resources. Without dedicated tests for this functionality, we cannot safely refactor or build upon the state management system.

---
## 2. Decision

We will create a new integration test suite focused exclusively on validating the end-to-end lifecycle of stateful resources. This will extend the mock injection pattern to **asset handlers** (`CreateFn`, `DestroyFn`), allowing for full control over the resource lifecycle in a test environment.

The core of this test will be a new HCL grid file that defines a workflow around a hypothetical, in-memory, stateful "counter" resource.

#### Test Mechanism & Mocks

1.  **Mock Asset**: We will simulate a `local_counter` asset. This asset is ideal as it has a simple state (an integer) and requires no external dependencies.

2.  **Mock Asset Handlers**:
    * The **`Create`** handler will be a "spy" that records its invocation and returns a new, thread-safe counter object instance.
    * The **`Destroy`** handler will also be a "spy" that simply records that it was called.

3.  **Mock Step Handler**:
    * A mock `counter_op` step handler (e.g., for an "increment" action) will receive the injected counter object via its `Deps` struct. It will verify it received a valid object and then call a method on it (e.g., `Increment()`).

#### Assertions

The integration test will run this HCL grid and assert the following conditions:

* **Singleton Creation**: The mock `Create` handler for the resource is called **exactly once**.
* **Instance Sharing**: Each step that `uses` the resource receives the **exact same object instance**, proving that the resource is correctly shared.
* **State Persistence**: The state of the resource is correctly modified across steps (e.g., if two "increment" steps run, the counter's final value is 2).
* **Guaranteed Destruction**: The mock `Destroy` handler is called **exactly once** after all dependent steps have completed.

---
## 3. Consequences

#### Pros
* **Validates Core Stateful Logic**: This provides high confidence in the entire resource management pipeline, from creation and injection to state sharing and cleanup.
* **Enables Safe Refactoring**: With a strong test safety net, we can confidently optimize or refactor the resource management and DAG execution logic in the future.
* **Provides a Developer Blueprint**: The test will serve as a clear, canonical example for developers on how to build and test runners that correctly interact with stateful resources.
* **Completes Core Engine Coverage**: Paired with the stateless tests in `ADR-004`, this brings us significantly closer to comprehensive test coverage of the entire execution engine.

#### Cons
* **Increased Test Complexity**: Testing stateful lifecycles and concurrency is inherently more complex than testing simple data flow, requiring more careful test setup.
* **Potential for Flakiness**: While mitigated by using simple, in-memory mocks, tests involving state can be more susceptible to race conditions if not architected carefully.