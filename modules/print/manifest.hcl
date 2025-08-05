runner "print" {
  description = "A general-purpose runner that prints any given value to the logs for debugging."

  input "input" {
    type        = any
    description = "The value to be printed. Can be of any type."
  }

  lifecycle {
    on_run = "OnRunPrint"
  }
}