# ADR-013: Dynamic Step Instancing with `for_each`

**Date**: 2025-08-05  
**Status**: Draft

## Context

`ADR-012` introduces step instancing via `count`, which solves the problem of creating a *statically known* number of instances. However, it does not address the use case where the number of instances is *dynamic* and depends on the output of another step. For example, a user might want to fetch a list of servers from an inventory step and then run a configuration step for each server in the list.

This requires a mechanism to create instances by iterating over a collection (a list or map) resolved at runtime.

## Decision

We will introduce a new meta-argument, `for_each`, to the `step` block.

1. **The `for_each` Meta-Argument**  
   A `step` block may specify `for_each = <COLLECTION>`, where `<COLLECTION>` is a reference to a list or a map. The DAG builder will create one step instance for each element in the collection. The `count` and `for_each` arguments are mutually exclusive.

2. **The `each` Variable**  
   Within a step instance created by `for_each`, the special variable `each` will be available:
   * `each.key`: The instance's corresponding map key or list index.
   * `each.value`: The instance's corresponding map value or list element.

3. **Instance Referencing**  
   To access instance outputs:
   * **Key Lookup**: `step.my_step.foo["key_name"]` will access the output of the instance corresponding to that key.
   * **Splat (`[*]`)**: `step.my_step.foo[*].output` will return a list of outputs, similar to its behavior with `count`.

An example of the feature:
```hcl
# step.inventory.users.output might be a map: { "alice": { id: 1 }, "bob": { id: 2 } }
step "user_setup" "create_home_dir" {
  for_each = step.inventory.users.output

  arguments {
    # each.key will be "alice", "bob"
    # each.value will be the corresponding map object
    username = each.key
    user_id  = each.value.id
  }
}

## Consequences

### Positive
- Enables fully dynamic workflows, a very powerful and frequently requested feature in configuration-as-code systems.  
- Unlocks the ability to compose steps in a much more flexible way, where one step generates work for another.  
- Further reduces boilerplate for complex, dynamic environments.

### Negative / Impact
- Introduces significant complexity to the DAG builder. The graph's shape is no longer known at parse time; it must be partially evaluated to resolve `for_each` expressions before being fully expanded.  
- Error handling becomes more complex. We need clear error messages for when a `for_each` expression evaluates to a non-collection type.  
- The user's mental model becomes more advanced. Clear documentation is critical.