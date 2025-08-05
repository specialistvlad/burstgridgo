runner "env_vars" {
  description = "Reads environment variables with support for filtering, defaults, and validation."

  # --- Inputs ---

  input "include" {
    type        = list(string)
    description = "A list of variable names to include in the output. If omitted, all variables are considered (unless 'prefix' is used)."
    default     = []
  }

  input "required" {
    type        = list(string)
    description = "A list of variable names that must be present in the environment or have a default. If any are missing, the step will fail."
    default     = []
  }

  input "defaults" {
    type        = map(string)
    description = "A map of default values to use if a variable is not found in the environment."
    default     = []
  }

  input "prefix" {
    type        = string
    description = "Only include variables that start with this prefix. Overridden by 'include' if both are provided."
    default     = ""
  }

  input "strip_prefix" {
    type        = bool
    description = "If true, the prefix is removed from the keys in the output map."
    default     = false
  }

  # --- Output ---

  output "vars" {
    type        = map(string)
    description = "A map containing the final set of environment variables after all processing."
  }

  # --- Lifecycle ---

  lifecycle {
    on_run = "OnRunEnvVars"
  }
}