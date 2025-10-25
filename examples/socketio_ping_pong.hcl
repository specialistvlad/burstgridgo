# File: examples/socketio_ping_pong.hcl
# This example demonstrates a single Socket.IO ping-pong request.
# It connects, emits a "ping" event, and waits for a "pong" event.
# To run this example use a command like: `make run ./examples/socketio_ping_pong.hcl`
# You may need to set SOCKETIO_WSS_URL or rely on the default.
step "env_vars" "read_env_vars_ping_pong" {
  arguments {
    required = [] # No required vars, just use defaults
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