# modules/counter_op/manifest.hcl
runner "counter_op" {
  description = "Performs an operation on a local_counter resource."

  # Declares that this runner needs a dependency named "counter" that
  # provides the "local_counter" asset type.
  uses "counter" {
    asset_type = "local_counter"
  }

  input "action" {
    type        = string
    description = "The action to perform: 'increment' or 'get'."
  }

  output "value" {
    type        = number
    description = "The value of the counter after a 'get' action."
  }

  lifecycle {
    on_run = "OnRunCounterOp"
  }
}