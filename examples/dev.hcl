module "env" {
  runner = "env_vars"
}

module "display_envs" {
  runner = "print"
  input = module.env.all
}

module "first_request" {
  runner = "http-request"
  url    = module.env.all.EVENT_BRIDGE_URL
}