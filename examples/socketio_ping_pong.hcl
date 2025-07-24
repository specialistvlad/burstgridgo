step "env_vars" "env" {}

step "socketio" "ping_pong" {
  arguments {
    url                  = step.env_vars.env.output.all.SOCKETIO_WSS_URL
    on_event             = "pong"
    emit_event           = "ping"
    timeout              = "5s"
    insecure_skip_verify = true
  }
}