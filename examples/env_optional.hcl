# File: examples/env_optional.hcl
# This example demonstrates how to use environment variables with default values.
# To run this example use a command like: `make run ./examples/env_optional.hcl`
# If LOG_LEVEL is not set, it will default to "info".
step "env_vars" "optional" {
  arguments {
    defaults = {
      "LOG_LEVEL" = "info"
    }
  }
}

step "print" "show_optional_config" {
  arguments {
    input = step.env_vars.optional.output.vars.LOG_LEVEL
  }
}