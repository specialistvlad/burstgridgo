# Make HTTP requests with dependencies
module "first_request" {
  runner = "http-request"
  url    = "https://httpbin.org/get"
}

# Make additional HTTP requests that depend on the first request
module "second_request" {
  runner     = "http-request"
  url        = "https://httpbin.org/delay/1" // This will take 1 second
  depends_on = ["first_request"]
}

# Make a third HTTP request that depends on the second request
module "third_request" {
  runner     = "http-request"
  url        = "https://httpbin.org/delay/2" // This will take 2 seconds
  depends_on = ["first_request"]
}

# Make a final HTTP request that depends on the second and third requests
module "final_request" {
  runner     = "http-request"
  url        = "https://httpbin.org/post"
  method     = "POST"
  depends_on = ["second_request", "third_request"]
}
