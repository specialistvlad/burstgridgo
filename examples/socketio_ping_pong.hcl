step "env_vars" "read_env_vars_ping_pong" {
  arguments {
    required = ["SOCKETIO_WSS_URL"]
    defaults = {
      SOCKETIO_WSS_URL = "wss://example.com/socket.io"
    }
  }
}

step "socketio" "ping_pong" {
  arguments {
    url                  = step.env_vars.read_env_vars_ping_pong.output.vars.SOCKETIO_WSS_URL
    emit_event           = "ping"
    on_event             = "pong"
    timeout              = "5s"
    insecure_skip_verify = true
    emit_data = {
      message = "Hello, World!"
    }
  }
}