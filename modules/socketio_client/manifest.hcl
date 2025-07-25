asset "socketio_client" {
  description = "Provides a shared, persistent Socket.IO client connection."

  input "url" {
    type        = string
    description = "The URL of the Socket.IO server (e.g., 'wss://host/path')."
  }

  input "namespace" {
    type        = string
    description = "The namespace to connect to."
    optional    = true
    default     = "/"
  }

  input "insecure_skip_verify" {
    type        = bool
    description = "If true, TLS certificate verification is skipped."
    optional    = true
    default     = false
  }

  lifecycle {
    create  = "CreateSocketIOClient"
    destroy = "DestroySocketIOClient"
  }
}