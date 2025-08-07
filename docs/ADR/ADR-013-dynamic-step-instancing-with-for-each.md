# ADR-013: Dynamic Step Instancing with `for_each`

**Date**: 2025-08-06

**Status**: Draft

## Context

ADR-012 introduces step instancing via `count`, which solves the problem of creating a number of instances known at or before runtime. However, it does not address the use case where instances should correspond directly to the elements of a collection. For example, a user might want to fetch a list of servers from an inventory step and then run a configuration step for each server.

This requires a mechanism to create instances by iterating over a map or a set. This ADR builds upon the instancing framework defined in ADR-012 to provide this capability.

## Decision

We will introduce a new meta-argument, `for_each`, to the `step` block.

1.  **The `for_each` Meta-Argument**
    The `for_each` argument accepts a map or a set of strings. Its presence immediately places the step in `Instanced Mode`, as defined in ADR-012. The `count` and `for_each` meta-arguments are mutually exclusive.

2.  **The `each` Variable**
    Within a step instance created by `for_each`, the special variable `each` will be available in the expression context. It is an object containing two attributes:
    * `each.key`: The instance's corresponding map key or set element value.
    * `each.value`: The instance's corresponding map value or set element value. (For a set, `each.key` and `each.value` are identical).

3.  **Instance Referencing Rules**
    As the step is in `Instanced Mode`, shorthand referencing is disallowed. Access must be explicit:
    * **Key Lookup**: `step.my_step.foo["key_name"]` will access the output of the instance corresponding to that key.
    * **Splat (`[*]`)**: `step.my_step.foo[*].output` will return a list containing the `output` attribute from all instances. Note: Since the source collection is a map or set, the order of elements in the resulting list is not guaranteed.

An example illustrates this. An inventory step could produce a map of users. A subsequent step, `user_setup.create_home_dir`, could set its `for_each` argument to reference this map. This would place it in `Instanced Mode` and create one instance per user. Within its arguments, it could use `each.key` as the username and `each.value.id` to get the user's ID, creating a unique home directory for each.

## Consequences

### Positive
* Enables fully dynamic workflows based on collection data.
* Unlocks powerful and flexible composition of steps, where one step generates work for another.
* Drastically reduces configuration boilerplate for complex, dynamic environments.

### Negative
* The use of `for_each` makes DAGs inherently dynamic. The resolution of the collection and validation of instance keys must be deferred to runtime, which can lead to later error discovery.

## Implementation Plan

### DAG Construction
The `for_each` expression must be evaluated to determine the collection of instances to create. This will almost always follow the "Dynamic Path" of DAG construction, where the step is represented as a placeholder and expanded by the executor at runtime. The "Static Path" could only be used if the `for_each` expression is a literal map or set value.

### Executor Design
The design principle from ADR-012 applies directly. The executor's logic should remain simple and uniform. The DAG builder/planner is responsible for the complexity of resolving the `for_each` collection and expanding the step. The executor will receive a collection of instances to run (keyed by the map/set keys) and will not need special logic compared to executing `count`-based instances.

### Expression Engine
The expression evaluation engine must be made aware of the `each` object (`each.key` and `each.value`) and inject it into the context when processing arguments for a `for_each`-instanced step.

### Validation Logic
The validation system must be updated to handle `for_each`-specific cases:
* A step cannot contain both `count` and `for_each` meta-arguments.
* The `for_each` expression must evaluate to a map or a set of strings.
* Any attempt to use shorthand referencing on the step must be flagged as an error.