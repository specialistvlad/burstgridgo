asset "local_counter" {
  description = "A simple, stateful, in-memory counter for testing the resource lifecycle."

  # This asset has no user-configurable arguments.

  lifecycle {
    create  = "CreateCounter"
    destroy = "DestroyCounter"
  }
}