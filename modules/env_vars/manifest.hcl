runner "env_vars" {
  description = "Reads all environment variables and provides them as an output."

  output "all" {
    type        = map(string)
    description = "A map of all environment variables."
  }

  lifecycle {
    on_run = "OnRunEnvVars"
  }
}