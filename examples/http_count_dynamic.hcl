# File: examples/http_count_dynamic.hcl
# This example demonstrates a dynamic fan-out pattern without a fan-in.
# The number of parallel requests is set by the REQUEST_COUNT env var.
# Note: This example does not have a "fan-in" step to collect results.
# To run this example use a command like: `REQUEST_COUNT=5 make run ./examples/http_count_dynamic.hcl`

# 1. Define a stateful, shared resource.
resource "http_client" "shared" {
  arguments {
    timeout = "45s"
  }
}

# 2. Define a dynamic provider for the count using environment variables.
# This allows the count to be configured at runtime via `export REQUEST_COUNT=...`
step "env_vars" "config" {
  arguments {
    # Default to 10 if not set
    defaults = {
      "REQUEST_COUNT" = "10"
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