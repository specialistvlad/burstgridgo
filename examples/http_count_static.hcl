# File: examples/http_count_static.hcl
# This example demonstrates a static fan-out with a `count` block.
# It runs 10 parallel requests and then has a final step that
# explicitly depends on a subset of those requests.
# To run this example use a command like: `make run ./examples/http_count_static.hcl`

# 1. Define a stateful, shared resource.
resource "http_client" "shared" {
  arguments {
    timeout = "45s"
  }
}

# 2. Define steps that consume the shared resource.
step "http_request" "first" {
  uses {
    client = resource.http_client.shared
  }
  arguments {
    url = "https://httpbin.org/get"
  }
}

# 3. These two steps are now replaced by a single block.
step "http_request" "delay_requests" {
  count = 10

  uses {
    client = resource.http_client.shared
  }
  arguments {
    url = "https://httpbin.org/delay/${count.index + 1}"
  }
  depends_on = ["http_request.first"]
}


# 4. The "fan-in" step now depends on the specific instances.
step "http_request" "final" {
  uses {
    client = resource.http_client.shared
  }
  arguments {
    url    = "https://httpbin.org/post"
    method = "POST"
  }
  # This is the key: we explicitly depend on each instance.
  depends_on = [
    "http_request.delay_requests[0]",
    "http_request.delay_requests[1]",
    "http_request.delay_requests[7]",
  ]
}