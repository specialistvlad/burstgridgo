module "health_check" {
  runner = "http-request"

  method = "GET"
  url    = "http://host.docker.internal:15060/engine-worker-api/health-check"

  expect {
    status = 200
  }
}
