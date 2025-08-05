# Useful for debugging, but less safe for production.
step "env_vars" "all_vars_for_debug" {}

step "print" "show_config" {
  arguments {
    input = step.env_vars.all_vars_for_debug.output.vars
  }
}
