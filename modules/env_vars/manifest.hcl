runner "env_vars" {
  description = "Reads environment variables with support for filtering, defaults, and validation."

  # --- Inputs ---

  input "include" {
    type        = list(string)
    description = "An explicit list of variable names to process. This list is combined with keys from the 'defaults' and 'required' blocks."
    default     = []
  }

  input "required" {
    type        = list(string)
    description = "A list of variable names that must be present. The step fails if a key is not in the environment and has no default. Keys listed here are automatically included for processing."
    default     = []
  }

  input "defaults" {
    type        = map(string)
    description = "A map of default values for variables not found in the environment. All keys in this map are automatically included for processing."
    default     = {}
  }

  input "prefix" {
    type        = string
    description = "A prefix for discovering variables from the environment. This is only used if 'include', 'defaults', and 'required' are all empty."
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