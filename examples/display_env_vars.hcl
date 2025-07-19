module "all_env_vars" {
  runner = "env_vars"
}

module "display_envs" {
  runner = "print"

  # This expression creates an implicit dependency
  # on the "all_env_vars" module.
  input = module.all_env_vars.all
}