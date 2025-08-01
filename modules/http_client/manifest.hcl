# asset "http_client" defines a shared, persistent HTTP client that can be
# reused across multiple steps for connection pooling and consistent configuration.
asset "http_client" {
  description = "Provides a shared, persistent HTTP client for connection reuse."

  input "timeout" {
    type        = string
    description = "Request timeout duration (e.g., '30s')."
    default     = "10s"
  }

  lifecycle {
    create  = "CreateHttpClient"
    destroy = "DestroyHttpClient"
  }
}

# runner "http_request" defines a stateless action that executes a single HTTP
# request using a shared http_client asset.
runner "http_request" {
  description = "Executes a simple HTTP request and returns the response."

  # uses declares that this runner requires an "http_client" asset. The key
  # "client" maps to the field name in the Go handler's Deps struct.
  uses "client" {
    asset_type = "http_client"
  }

  input "url" {
    type        = string
    description = "The URL to send the request to."
  }

  input "method" {
    type        = string
    description = "The HTTP method to use."
    default     = "GET"
  }

  output "status_code" {
    type        = number
    description = "The HTTP status code of the response."
  }

  output "body" {
    type        = string
    description = "The response body as a string."
  }

  lifecycle {
    on_run = "OnRunHttpRequest"
  }
}