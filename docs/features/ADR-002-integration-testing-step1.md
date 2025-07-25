# ADR-002: Integration Testing - Step 1 (Stateless Workflows)
- **Status**: Proposal
- **Author**: Vladyslav Kazantsev
- **Date**: 2025-07-24

This document outlines the first, iterative step in creating a robust integration testing strategy for the `burstgridgo` engine, focusing exclusively on stateless workflows.

---
## 1. Context & Problem Statement

The core of `burstgridgo` is its ability to orchestrate complex workflows defined in HCL. While unit tests are valuable for isolated logic, they cannot validate the complete end-to-end pipeline: HCL parsing -> DAG construction -> dependency resolution -> concurrent execution.

We need a testing strategy that is **fast**, **reliable**, and **isolated** from external network dependencies to ensure the core engine works as expected.

---
## 2. Proposed Solution: Deterministic Workflow Testing

We will adopt an iterative approach, beginning with the simplest, most common workflows. This first step will focus entirely on **validating the engine's orchestration of stateless `step` blocks** that do not involve `resource` management.

#### **Core Mechanism: Executor-Level Mock Injection**

To achieve test isolation without the risks of modifying global state, we will refactor the `dag.NewExecutor` to accept an optional map of handler overrides. A test will provide a map of handler names (e.g., "OnRunPrint") to their corresponding mock Go functions. The executor's worker will prioritize an override if one is provided, falling back to the global registry otherwise.

#### **Test Strategy: "Source and Spy"**

Our tests will treat HCL grid files as pure **topologies** that define the wiring between steps. We will then override the handlers at the start and end of this topology to create a fully controlled test environment.

* **The "Source" Mock**: For a workflow like `display_env_vars.hcl`, we will override the `OnRunEnvVars` handler. This mock will not read the actual OS environment but will instead produce a known, hardcoded data map. This makes the test's input deterministic.

* **The "Spy" Mock**: We will override the `OnRunPrint` handler. This mock will not print to the console but will instead capture its input into a variable that the test can access.

The test's assertion is then a simple, powerful comparison: verify that the data captured by the "Spy" is identical to the data produced by the "Source." This cleanly validates the engine's entire data-passing mechanism in isolation.

---
## 3. Implementation Plan

1.  Refactor the `dag.NewExecutor` constructor to accept the optional `overrides` map.
2.  Update the executor's internal worker logic to prioritize checking for and using a handler from the `overrides` map before consulting the global registries.
3.  Create a new test suite file, `main_integration_test.go`.
4.  Implement a test helper function to encapsulate the logic of parsing a grid file and running the executor with a given set of overrides.
5.  Implement the first test case for the `display_env_vars.hcl` workflow, creating both the "Source" mock for `env_vars` and the "Spy" mock for `print`.
6.  The test will assert that the "Spy" mock's captured input matches the "Source" mock's known output.

---
## 4. Consequences of This First Step

#### **Pros**
-   **Fully Deterministic**: By controlling both the input and output of the workflow, tests are completely predictable and free from environmental side effects.
-   **Immediate Value**: Quickly delivers a high-value test that validates the engine's core data-passing and dependency-sequencing logic.
-   **Solid Foundation**: The refactored `Executor` becomes the foundational piece for all subsequent, more complex integration tests.
-   **Low Risk & Complexity**: Starts with a simple workflow, making it easier to build and debug the test harness itself.

#### **Future Work (Explicitly Excluded from this Step)**
-   **Resource Lifecycle Testing**: This initial phase does not test the `uses` block, resource creation, or resource destruction. This will be the focus of "Step 2".
-   **External Call Mocking**: Does not involve mocking network calls, as this first test is fully self-contained.