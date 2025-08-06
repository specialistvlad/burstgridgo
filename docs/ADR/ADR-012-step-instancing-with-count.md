# ADR-012: Step Instancing with `count`

**Date**: 2025-08-05  
**Status**: Draft

## Context

Our current configuration processing model is static. Each `step` block defined in an HCL file corresponds to exactly one node in the execution DAG. This forces users to write repetitive configuration blocks if they need to perform the same action multiple times with only minor variations (e.g., calling an API with a different endpoint ID). This is verbose, error-prone, and hard to maintain. We need a way to express "run this step N times."

## Decision

We will introduce a new meta-argument, `count`, to the `step` block.

1. **The `count` Meta-Argument**  
   A `step` block may specify `count = N`, where `N` is a non-negative integer. Our DAG builder will expand this single block into `N` distinct step instances.

2. **The `count.index` Variable**  
   Within a step instance created by `count`, the special variable `count.index` will be available. It will hold the zero-based index of that instance (`0`, `1`, `2`, ...), allowing for unique arguments per instance.

3. **Instance Referencing**  
   To access the outputs of these instances, the following reference syntaxes will be supported:
   * **Index Lookup**: `step.my_step.foo[0]` will access the output of the first instance.
   * **Splat (`[*]`)**: `step.my_step.foo[*].output` will return a list containing the `output` attribute from all instances, preserving order. This is the primary mechanism for aggregation.

An example of the complete feature set:
```hcl
step "http_request" "ping" {
  count = 5

  arguments {
    url = "https://api.example.com/v1/status/${count.index}"
  }
}

step "print" "results" {
  arguments {
    # The splat expression collects all 5 status codes into a list
    codes = http_request.ping[*].output.status_code
  }
}
```

## Consequences

### Positive
- Significantly reduces configuration boilerplate for parallel, fixed-size tasks.  
- Makes configurations easier to read and maintain.  
- Introduces the foundational "instancing" logic into the DAG builder, paving the way for more advanced dynamic features.

### Negative / Impact
- The DAG builder's logic must be refactored to handle the one-to-many expansion of a step block.  
- The expression evaluation engine must be made aware of the `count.index` context variable.  
- New validation is required to handle invalid `count` values (e.g., negative numbers, non-integers) and incorrect references.
