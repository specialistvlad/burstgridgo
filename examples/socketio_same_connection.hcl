step "env_vars" "read_env" {}

# 1. Define a stateful resource for the persistent Socket.IO connection.
resource "socketio_client" "shared_connection" {
  arguments {
    url                  = step.env_vars.read_env.output.all.SOCKETIO_WSS_URL
    insecure_skip_verify = true
  }
}

# 2. Use the shared client to send the first request.
# The 'uses' block injects the live client object into the step's handler.
step "socketio_request" "ping_pong_1" {
  uses {
    client = resource.socketio_client.shared_connection
  }
  arguments {
    emit_event = "ping"
    on_event   = "pong"
    timeout    = "1s"
  }
}
# 3. Use the shared client to send the second request.
# The 'uses' block injects the live client object into the step's handler.
step "socketio_request" "ping_pong_2" {
  uses {
    client = resource.socketio_client.shared_connection
  }
  arguments {
    emit_event = "ping"
    on_event   = "pong"
    timeout    = "1s"
    emit_data  = {}
  }
}

# 4. Display the result of the last step.
step "print" "display_result1" {
  arguments {
    input = step.socketio_request.ping_pong_1.output.response_data
  }
}

step "print" "display_result2" {
  arguments {
    input = step.socketio_request.ping_pong_2.output.response_data
  }
}