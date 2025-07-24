# File: examples/http_request.hcl

step "http_request" "first" {
  arguments {
    url = "https://httpbin.org/get"
  }
}

step "http_request" "second" {
  arguments {
    url = "https://httpbin.org/delay/1"
  }
  depends_on = ["first"]
}

step "http_request" "third" {
  arguments {
    url = "https://httpbin.org/delay/2"
  }
  depends_on = ["first"]
}

step "http_request" "final" {
  arguments {
    url    = "https://httpbin.org/post"
    method = "POST"
  }
  depends_on = [
    "second",
    "third",
  ]
}