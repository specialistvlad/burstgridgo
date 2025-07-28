runner "socketio" {
  description = "Connects to a Socket.IO server, emits an event, and waits for a response event."

  input "url" {
    type        = string
    description = "The URL of the Socket.IO server."
  }
  input "namespace" {
    type        = string
    description = "The namespace to connect to."
    optional    = true
    default     = "/"
  }
  input "on_event" {
    type        = string
    description = "The name of the event to listen for as a success signal."
  }
  input "emit_event" {
    type        = string
    description = "The name of the event to emit after connecting."
    optional    = true
  }
  input "emit_data" {
    type        = any
    description = "The data payload to send with the emitted event."
    optional    = true
  }
  input "timeout" {
    type        = string
    description = "Timeout for the entire operation (e.g., '10s')."
    optional    = true
    default     = "10s"
  }
  input "insecure_skip_verify" {
    type        = bool
    description = "If true, TLS certificate verification is skipped."
    optional    = true
    default     = false
  }

  output "response_data" {
    type        = any
    description = "The data payload received from the on_event."
  }

  lifecycle {
    on_run = "OnRunSocketIO"
  }
}