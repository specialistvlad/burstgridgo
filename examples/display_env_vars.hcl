# Read all environment variables
module "all_env_vars" {
  runner = "env_vars"
}


# Display all environment variables
module "display_envs" {
  runner = "print"
  input = module.all_env_vars.all
}