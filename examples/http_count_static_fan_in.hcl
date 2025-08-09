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


# 4. This "fan-in" step collects and prints the output from ALL instances.
# It demonstrates the splat operator working on the dynamic group.
step "print" "show_all_results" {
  arguments {
    # This implicitly depends on the entire "delay_requests" group finishing.
    # The splat operator collects the 'output' from every instance into a list.
    input = step.http_request.delay_requests[*].output
  }
}