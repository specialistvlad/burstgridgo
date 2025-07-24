step "http_request" "first" {
  arguments {
    url = "https://httpbin.org/get"
  }
}

step "http_request" "second" {
  arguments {
    url = "https://httpbin.org/delay/1" # This will take 1 second
  }
  depends_on = [step.http_request.first]
}

step "http_request" "third" {
  arguments {
    url = "https://httpbin.org/delay/2" # This will take 2 seconds
  }
  depends_on = [step.http_request.first]
}

step "http_request" "final" {
  arguments {
    url    = "https://httpbin.org/post"
    method = "POST"
  }
  depends_on = [
    step.http_request.second,
    step.http_request.third,
  ]
}