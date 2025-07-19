module "first_request" {
  runner = "http-request"
  url    = "https://httpbin.org/get"
}

module "second_request" {
  runner     = "http-request"
  url        = "https://httpbin.org/delay/1" // This will take 1 second
  depends_on = ["first_request"]
}

# module "third_request" {
#   runner     = "http-request"
#   url        = "https://httpbin.org/delay/2" // This will take 2 seconds
#   depends_on = ["first_request"]
# }

# module "final_request" {
#   runner     = "http-request"
#   url        = "https://httpbin.org/post"
#   method     = "POST"
#   depends_on = ["second_request", "third_request"]
# }
