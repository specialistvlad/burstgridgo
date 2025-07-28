# This step calls the "env_vars" runner and gives this instance the name "read_env".
step "env_vars" "read_env" {}

# This step calls the "print" runner. It uses HCL interpolation to access the
# output of the "read_env" step above.
step "print" "display" {
  arguments {
    input = step.env_vars.read_env.output.all
  }
}