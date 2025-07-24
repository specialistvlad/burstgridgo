step "env_vars" "read_env" {}

step "socketio" "ping_pong" {
  arguments {
    url                  = step.read_env.output.all.SOCKETIO_WSS_URL
    on_event             = "pong"
    emit_event           = "ping"
    timeout              = "5s"
    insecure_skip_verify = true
  }
}