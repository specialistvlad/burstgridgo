# 1. Define a stateful, shared resource.
# This remains unchanged.
resource "http_client" "shared" {
  arguments {
    timeout = "45s"
  }
}

# 2. Define a dynamic provider for the count using environment variables.
# This allows the count to be configured at runtime via `export REQUEST_COUNT=...`
step "env_vars" "config" {
  arguments {
    # Default to 10 to ensure index [3] is always valid for the demo.
    defaults = {
      "REQUEST_COUNT" = "2"
    }
  }
}

# 3. Define the initial step in the main sequence.
step "http_request" "first" {
  uses {
    client = resource.http_client.shared
  }
  arguments {
    url = "https://httpbin.org/get"
  }
}

# 4. This step now has a dynamic count derived from the env_vars step.
step "http_request" "delay_requests" {
  # The count is now determined dynamically. We use tonumber() to convert the
  # environment variable string to a number.
  count = tonumber(step.env_vars.config.output.vars.REQUEST_COUNT)

  uses {
    client = resource.http_client.shared
  }
  arguments {
    url = "https://httpbin.org/delay/${count.index + 1}"
  }
  depends_on = ["step.http_request.first"]
}


# 5. This "fan-in" step collects and prints the output from ALL instances.
# It demonstrates the splat operator working on the dynamic group.
# step "print" "show_all_results" {
#   arguments {
#     # This implicitly depends on the entire "delay_requests" group finishing.
#     # The splat operator collects the 'output' from every instance into a list.
#     input = step.http_request.delay_requests[*].output
#   }
# }