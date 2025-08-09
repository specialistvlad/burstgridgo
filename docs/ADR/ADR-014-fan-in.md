# ADR-014: Fan-in for dynamic `count` and `for_each`

**Date**: 2025-08-08

**Status**: Draft

## Context


## Decision

## Consequences

### Positive
* Enables fully dynamic workflows based on collection data.
* Unlocks powerful and flexible composition of steps, where one step generates work for another.
* Drastically reduces configuration boilerplate for complex, dynamic environments.

### Negative


## Implementation Plan


#### Step x.x: Executor - Output Aggregation
* **Goal:** Ensure the collected results of the dynamic instances are correctly exposed to the rest of the graph as a single list output from the placeholder node.
* **What:**
    1.  **Aggregate Outputs:** In `internal/executor/worker.go`, after the instance loop finishes, the worker will aggregate the outputs from all `N` instances into a single list.
    2.  **Set Placeholder Node Output:** This final list becomes the `Output` of the main placeholder node, making the result available to any downstream steps.
* **Verification:**
    * Evolve the existing tests by adding new assertions. New test cases are added only for distinct error scenarios.
    1.  **Evolve `TestCoreExecution_Count_Dynamic`:** Add a downstream step and assert that it receives a *list* containing 3 outputs.
    2.  **Evolve `TestCoreExecution_Count_Dynamic_Zero`:** Add a downstream step and assert that it receives an *empty list*.
    3.  **Create new tests for error scenarios:**
        * `TestErrorHandling_Count_Dynamic_InvalidType`: Asserts the run fails with a clear type-mismatch error if `count` resolves to a non-numeric type.
        * `TestErrorHandling_Count_Dynamic_Negative`: Asserts the run fails with a clear error if `count` resolves to a negative number.