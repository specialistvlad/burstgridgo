# ADR-013: Architecturally Split the Executor Package

**Date**: 2025-08-08

**Status**: Accepted

## Context

The `executor` package, in its original form, was monolithic. It tightly coupled two distinct responsibilities:

1.  **Generic Concurrency**: The logic for managing a worker pool, traversing a Directed Acyclic Graph (DAG), handling dependencies, and managing concurrent execution state.
2.  **HCL-Specific Logic**: The logic for interpreting HCL-defined tasks, building HCL evaluation contexts (`buildEvalContext`), and managing the lifecycle of HCL-defined resources.

This tight coupling led to several issues:
* **Poor Separation of Concerns**: The core `Executor` struct contained fields specific to HCL (`registry`, `converter`), making it impossible to use for anything else.
* **Reduced Testability**: Testing the core concurrency logic required setting up a full HCL environment.
* **Low Reusability**: The valuable DAG execution engine could not be reused for other, non-HCL tasks.
* **Difficult Maintenance**: Changes to HCL logic were made in the same files that handled complex concurrency, increasing cognitive load and the risk of introducing bugs like race conditions.

## Decision

We will refactor the monolithic `executor` package by splitting it into two new, distinct packages with clear responsibilities, following the principle of Composition over Inheritance.

1.  **A new, generic `pkg/executor` will be created.**
    * **Responsibility**: This package will be a pure, concurrent DAG execution engine.
    * **Contract**: It will operate on a generic `dag.Runnable` interface (`interface { Run(ctx) error }`), making it completely agnostic to the type of tasks it runs.
    * **API**: It will provide a simple `New(graph, numWorkers)` constructor and an `Execute(ctx)` method.

2.  **The existing `executor` logic will be moved to a new `pkg/hclrunner` package.**
    * **Responsibility**: This package will contain all the HCL-specific logic. It will be the "specialization" layer.
    * **Implementation**: It will provide concrete implementations of the `dag.Runnable` interface (e.g., `resourceRunner`, `stepRunner`) that know how to execute HCL-defined tasks.
    * **Composition**: The main `hclrunner.Runner` will use the generic `pkg/executor` internally to run the graph.
    * **Public API**: The `hclrunner.New(...)` constructor will become the primary public entry point for users of this library. This is a deliberate change to the API's location to enforce the new architecture.

## Consequences

### Positive
* **Strong Separation of Concerns**: The generic engine is now fully decoupled from the HCL implementation details.
* **High Reusability**: The `pkg/executor` engine is now a library-quality component that can be reused to run any kind of DAG-based workflow.
* **Improved Testability**: The `executor` can be tested with simple mock `Runnable` tasks, and the `hclrunner` can be tested for its logic in isolation.
* **Clearer Mental Model**: Developers can work on the engine and the HCL logic independently, reducing complexity and improving productivity.

### Negative
* **API Location Change**: Existing consumers of this library will need to update their import paths from `.../executor` to `.../hclrunner`. This has been deemed an acceptable trade-off for the architectural benefits.
* **Increased Indirection**: The split introduces more interfaces and packages, which adds a layer of indirection. This is a standard trade-off for achieving modularity.