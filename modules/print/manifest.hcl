runner "print" {
  description = "Prints a given HCL value to the console."

  input "input" {
    type        = any
    description = "The value to be printed."
  }

  lifecycle {
    on_run = "OnRunPrint"
  }
}