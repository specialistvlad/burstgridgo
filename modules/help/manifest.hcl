runner "help" {
  description = "Displays the command-line help text."

  lifecycle {
    on_run = "OnRunHelp"
  }
}