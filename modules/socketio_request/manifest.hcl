runner "socketio_request" {
  description = "Emits an event on a shared Socket.IO connection and waits for a response."

  uses "client" {
    asset_type = "socketio_client"
  }

  input "on_event" {
    type        = string
    description = "The name of the event to listen for as a success signal."
  }

  input "emit_event" {
    type        = string
    description = "The name of the event to emit after connecting."
  }

  input "emit_data" {
    type        = any
    description = "The data payload to send with the emitted event."
    optional    = true
  }

  input "timeout" {
    type        = string
    description = "Timeout for this specific request (e.g., '5s')."
    optional    = true
    default     = "10s"
  }

  output "response_data" {
    type        = any
    description = "The data payload received from the on_event."
  }

  lifecycle {
    on_run = "OnRunSocketIORequest"
  }
}