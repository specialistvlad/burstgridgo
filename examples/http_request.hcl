# File: examples/http_request.hcl

# 1. Define a stateful, shared resource.
# This creates a single, persistent HTTP client that can be reused.
resource "http_client" "shared" {
  arguments {
    timeout = "45s"
  }
}

# 2. Define steps that consume the shared resource.
step "http_request" "first" {
  # Inject the shared client into the runner.
  uses {
    client = resource.http_client.shared
  }
  arguments {
    url = "https://httpbin.org/get"
  }
}

step "http_request" "second" {
  uses {
    client = resource.http_client.shared
  }
  arguments {
    url = "https://httpbin.org/delay/1"
  }
  # Use the full address "runner_type.instance_name" for clarity.
  depends_on = ["http_request.first"]
}

step "http_request" "third" {
  uses {
    client = resource.http_client.shared
  }
  arguments {
    url = "https://httpbin.org/delay/2"
  }
  depends_on = ["http_request.first"]
}

step "http_request" "final" {
  uses {
    client = resource.http_client.shared
  }
  arguments {
    url    = "https://httpbin.org/post"
    method = "POST"
  }
  depends_on = [
    "http_request.second",
    "http_request.third",
  ]
}