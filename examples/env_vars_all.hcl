# File: examples/env_vars_all.hcl
# This example demonstrates how to capture all environment variables.
# This is useful for debugging but not recommended for production.
# To run this example use a command like: `make run ./examples/env_vars_all.hcl`
step "env_vars" "all_vars_for_debug" {}

step "print" "show_all_vars_for_debug" {
  arguments {
    input = step.env_vars.all_vars_for_debug.output.vars
  }
}