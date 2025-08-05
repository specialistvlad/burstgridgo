# This step will fail if DB_PASS is not set in the environment.
step "env_vars" "db" {
  arguments {
    prefix       = "DB_" // Use 'prefix' to specify WHICH variables to find.
    strip_prefix = true  // Use 'strip_prefix' to enable stripping that prefix from the output.
    defaults = {
      "DB_HOST" = "localhost"
      "DB_USER" = "guest"
    }
    required = ["DB_HOST", "DB_USER", "DB_PASS"]
  }
}

step "print" "show_config" {
  arguments {
    input = step.env_vars.db.output.vars
  }
}
