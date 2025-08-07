# ADR-012: Step Instancing with `count`

**Date**: 2025-08-06
**Status**: Implementation

## Context

Our current configuration processing model is static. Each `step` block defined in an HCL file corresponds to exactly one node in the execution DAG. This forces users to write repetitive configuration blocks if they need to perform the same action multiple times with only minor variations. This is verbose, error-prone, and hard to maintain. We need a way to express "run this step N times."

## Decision

We will introduce a new meta-argument, `count`, to the `step` block, governed by a system of instancing modes and strict referencing rules.

1.  **Instancing Modes**
    A step operates in one of two modes: `Singular` or `Instanced`. The mode is determined by the explicit presence of an instancing meta-argument.
    * **Singular Mode**: This is the default when no instancing keyword (such as `count`) is specified. The step is treated as having exactly one instance.
    * **Instanced Mode**: This mode is activated when an instancing meta-argument (such as `count`) is explicitly present in the configuration. This makes the step's multiplicity explicit and is forward-compatible with other instancing keywords like `for_each`.

2.  **The `count.index` Variable**
    Within any instance created when `count` is used, the special variable `count.index` will be available. It will hold the zero-based integer index of that instance, allowing for unique arguments per instance.

3.  **Instance Referencing Rules**
    Accessing step outputs is governed by the step's instancing mode.
    * **Singular Mode Access**: For convenience, steps in `Singular` mode allow shorthand referencing. A reference like `step.my_step.foo` will directly access the output of the single instance.
    * **Instanced Mode Access**: Shorthand referencing is disallowed to prevent ambiguity. Access must be explicit through either index lookup (e.g., `step.my_step.foo[0]`) or a splat expression (e.g., `step.my_step.foo[*].output`). This rule applies to any step in `Instanced Mode`.
    * Any attempt to use shorthand referencing on a step in `Instanced Mode` will result in a validation error.

An example illustrates this. A user could define a step named `http_request.ping` and set its `count` to 5, placing it in `Instanced Mode`. Within that step's arguments, they could use `count.index` to construct a unique URL for each instance. A separate step, such as `print.results`, could then aggregate all five status codes into a list by referencing `http_request.ping[*].output.status_code`.

## Consequences

### Positive
* Significantly reduces configuration boilerplate for parallel, fixed-size tasks.
* Makes configurations easier to read and maintain.
* Introduces a robust instancing model into the core engine, paving the way for more advanced dynamic features like `for_each`.

### Negative
* For steps utilizing a dynamic `count`, validation of instance-specific references is necessarily deferred from parse-time to runtime. This can result in later error discovery compared to statically-defined steps.

## Implementation Recommendations

### DAG Construction
The DAG builder must be upgraded to support two modes of construction. During the configuration analysis phase, the engine will attempt to evaluate the `count` expression.
* **Static Path**: If the expression resolves to an integer at parse time (i.e., it has no runtime dependencies), the step is fully expanded into its instances, and the DAG is constructed statically. This allows for comprehensive upfront validation of instance references.
* **Dynamic Path**: If the expression depends on other step outputs, the step's expansion is deferred until runtime. It is represented as an unexpanded placeholder in the initial DAG and resolved by the executor.

### Executor Design
To ensure a clean separation of concerns, the complexity of the static vs. dynamic expansion must be contained entirely within the DAG building and planning phase. The core executor must be designed to operate on a uniform collection of step instances, regardless of how they were generated. This simplifies the execution logic and improves system robustness.

### Expression Engine
The expression evaluation engine must be made aware of the `count.index` context variable when processing arguments for an instanced step.

### Validation Logic
The validation system must be updated to handle several new cases: invalid `count` values (e.g., negative numbers, non-integers), incorrect use of shorthand referencing on `Instanced` mode steps, and index-out-of-bounds errors.

## Implementation Plan

This plan outlines a safe, incremental approach to implementing the `count` feature as defined in ADR-012. Each phase and step is designed to be a small, verifiable change that leaves the system in a stable, fully-tested state, validated by the existing integration test suite.

### Phase 1: Preparatory Refactoring

**Goal:** Introduce the new data structures and concepts required for instancing into the codebase without changing any existing behavior.

#### Step 1.1: Add `count` and Instancing Fields to Data Models
> **Status: ✅ Implemented.** The `config.Step` and `hcl.Step` structs have been updated, and the translation layer correctly sets the `InstancingMode`.

* **Why:** To establish the foundational data structures in the core `config` and `schema` packages.
* **What:**
    1.  In `internal/hcl/hcl_schema.go`, add `Count hcl.Expression hcl:"count,optional"` to the `schema.Step` struct.
    2.  In `internal/config/model.go`, define the `InstancingMode` enum (`ModeSingular`, `ModeInstanced`).
    3.  In `internal/config/model.go`, add `Count hcl.Expression` and `Instancing InstancingMode` to the `config.Step` struct.
    4.  In `internal/hcl/translate_model.go`, update `translateStep` to copy the `Count` expression and to default `Instancing = config.ModeSingular`.
* **Verification:** All existing integration tests must pass.

#### Step 1.2: Create an "Expansion" Seam in the DAG Builder
> **Status: ⚠️ Deviated.** This step was not performed as planned. Instead of creating a trivial seam, the implementation jumped ahead and combined this refactoring with the feature logic from Phase 3.1.

* **Why:** To create a dedicated entry point for the future one-to-many expansion logic, separating it from the main graph construction code.
* **What:**
    1.  In `internal/dag/build.go`, introduce a new internal helper function: `expandStep(s *config.Step) []*config.Step`.
    2.  The initial implementation will be trivial, returning the input step in a single-element slice: `return []*config.Step{s}`.
    3.  The main loop in `createNodes` will be refactored to call `expandStep` and iterate over the slice it returns.
* **Verification:** This is a behavior-neutral refactoring. All integration tests must pass.

### Phase 2 (Revised): Unify the Internal Model via Test-Driven Refactoring

**Goal:** To refactor the application's core to treat every step as an instance, creating a unified internal model. This is done safely by refactoring the tests to be resilient to the internal changes first.

> **Status: ✅ Implemented.** Based on the code in `internal/dag/build.go` and the `internal/dag/links_*.go` files, the core internal model has been successfully refactored to use indexed IDs for all steps.

#### Step 2.1: Refactor Integration Tests for Resilience
* **Context:** Our analysis confirmed that current tests, like `TestCLI_MergesHCL_FromDirectoryPath`, rely on asserting against hardcoded node ID formats in log output (e.g., `step=step.print.step_A`).
* **Why:** To decouple tests from the specific format of internal node IDs, ensuring they validate *behavior* only.
* **What:**
    1.  Analyze the `integration_tests/` suite and identify all assertions that rely on hardcoded node ID formats.
    2.  Introduce new helper functions in the `internal/testutil` package, such as `testutil.AssertStepRan(t, result, "runner_type", "step_name")`.
    3.  Refactor all brittle tests to use these new abstract helper functions.
* **Verification:** All tests must pass after this test-only refactoring.

#### Step 2.2: Introduce Indexed Node IDs
* **Context:** Our analysis confirmed that `internal/dag/build.go`'s `createNodes` function currently generates simple, non-indexed IDs (e.g., `step.print.A`).
* **Why:** To make the node identity explicitly instanced from the moment of creation.
* **What:** In `internal/dag/build.go`, change the node ID generation logic to produce IDs with an index suffix for all steps (e.g., `step.runner.name[0]`).

#### Step 2.3: Update Dependency Linking Logic
* **Context:** Our analysis confirmed that the logic in `internal/dag/links_*.go` resolves dependencies like `"runner.A"` by looking for a non-indexed node ID.
* **Why:** To teach the internal linking code how to resolve dependencies to the new indexed node IDs.
* **What:** In `internal/dag/links_explicit.go` and `internal/dag/links_implicit.go`, update `linkExplicitDeps` and `linkImplicitDeps` to correctly translate a reference like `"runner.A"` into a lookup for the `...[0]` node in the graph.

#### Step 2.4: Update the Test Harness Helper
* **Why:** To align the single point of truth in the test suite with the new internal node ID format.
* **What:** Update the internal logic of the helper functions created in Step 2.1 to now construct and look for the new `...[0]` ID format.

#### Step 2.5: Final Verification
* **Why:** To prove that the core application refactoring was successful and did not alter any externally-observable behavior.
* **Verification:** Run the entire test suite. No test changes should be required in this step. All tests must pass.

### Phase 3: Activate the `count` Feature

**Goal:** To implement the user-facing `count` functionality, building upon the refactored foundation.

#### Step 3.1: Implement Static `count` Expansion
> **Status: ✅ Implemented.** `internal/hcl/translate_model.go` correctly sets the instancing mode, and `internal/dag/build.go` correctly expands steps with static `count` values.

* **Why:** To deliver the simplest version of the feature first and validate the core instancing logic.
* **What:**
    1.  Update `internal/hcl/translate_model.go` to set `InstancingMode` to `ModeInstanced` if the `count` keyword is present.
    2.  Update the `dag/expandStep` helper to evaluate static `count` values and generate `N` instances.

#### Step 3.2: Inject the `count.index` Variable
> **Status: ✅ Implemented.** `internal/executor/context_builder.go` correctly parses the instance index from the node ID and injects the `count.index` variable into the evaluation context.

* **Why:** To make the instance index available to expressions, as required by the ADR.
* **What:** Update `internal/executor/context_builder.go` to parse the index from a node's ID and inject the `count` object into the `hcl.EvalContext`.

#### Step 3.3: Enforce `Instanced Mode` Referencing Rules
> **Status: ✅ Implemented.** The dependency linking and HCL context builder are now fully instance-aware.

* **Why:** To implement strict access control and prevent ambiguous references.
* **What:**
    1.  Teach `internal/dag/links_explicit.go` and `internal/dag/links_implicit.go` to resolve an indexed dependency access (e.g., `step.foo.bar[1]`).
    > **Note:** The linkers now correctly parse both explicit `depends_on` strings and implicit HCL traversals to identify specific instance indices.
    2.  Update the DAG linking logic to return a validation error for any shorthand access to a step in `Instanced Mode`.
    > **Note:** This validation is now implemented for both implicit and explicit dependencies, preventing ambiguous graph states.
    3.  Update `internal/executor/context_builder.go` to expose outputs from instanced steps as lists in the HCL evaluation context.

#### Step 3.4: Implement Dynamic `count`
> **Status: ❌ Not Implemented.** The entire dynamic evaluation path is missing.

* **Why:** to support `count` values that depend on the output of other steps.
* **What:**
    1.  Update `internal/dag/build.go` to create a special "unexpanded" placeholder node for steps with a dynamic `count`.
    > **Note:** This is not implemented. `internal/dag/build.go` incorrectly treats a step with a dynamic count as a single instance.
    2.  Update `internal/executor/worker.go` to recognize these placeholders, evaluate the `count` expression at runtime, and dynamically expand the step into its final nodes.
    > **Note:** This is not implemented. The executor is unaware of placeholders or runtime expansion.

#### Testing Strategy for Phase 3
> **Status: ⚠️ Partially Implemented.** The core referencing rules are now well-tested, but tests for dynamic `count` are still pending.

* **New Features:** A genuinely new capability, such as the initial static `count` or the dynamic `count` feature, justifies a new test file (e.g., `integration_tests/core_execution/static_count_test.go`, `TestCoreExecution_Count_Dynamic`).
* **Feature Enhancements:** An enhancement to an existing feature, such as adding `count.index` support or `depends_on` with an index, should expand the existing relevant test file with new assertions.
    > **Note:** A new test, `integration_tests/core_execution/instancing_test.go`, has been added to validate successful indexed dependency resolution.
* **Error Cases:** A test that validates a failure condition, such as attempting shorthand access on an instanced step, justifies a new, focused test in the `integration_tests/error_handling/` test suite.
    > **Note:** A new test, `integration_tests/error_handling/ambiguous_dependency_test.go`, has been added to validate failure cases for both implicit and explicit ambiguous dependencies.