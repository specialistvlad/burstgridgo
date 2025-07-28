runner "print" {
  description = "Prints the contents of a map to the console."

  input "input" {
    type        = map(string)
    description = "The map of strings to be printed."
  }

  lifecycle {
    on_run = "OnRunPrint"
  }
}