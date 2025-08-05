
# If LOG_LEVEL is not set, it will default to "info".
step "env_vars" "optional" {
  arguments {
    defaults = {
      "LOG_LEVEL" = "info"
    }
  }
}

step "print" "show_config" {
  arguments {
    input = step.env_vars.optional.output.vars.LOG_LEVEL
  }
}
