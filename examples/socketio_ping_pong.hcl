module "env" {
  runner = "env_vars"
}

module "socketio_ping_pong" {
  runner               = "socketio"
  url                  = module.env.all.SOCKETIO_WSS_URL
  on_event             = "pong"
  emit_event           = "ping"
  timeout              = "5s"
  insecure_skip_verify = true
}