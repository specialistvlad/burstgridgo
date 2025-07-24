# This step calls the "env_vars" runner and gives this instance the name "read".
step "env_vars" "read" {}

# This step calls the "print" runner. It uses HCL interpolation to access the
# output of the "read" step above.
step "print" "display" {
  arguments {
    input = step.env_vars.read.output.all
  }
}