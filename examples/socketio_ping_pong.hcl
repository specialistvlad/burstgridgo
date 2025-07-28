step "env_vars" "read_env" {}

step "socketio" "ping_pong" {
  arguments {
    url                  = step.env_vars.read_env.output.all.SOCKETIO_WSS_URL
    emit_event           = "ping"
    on_event             = "pong"
    timeout              = "5s"
    insecure_skip_verify = true
  }
}