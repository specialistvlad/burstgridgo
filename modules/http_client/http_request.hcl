runner "http_request" {
  description = "Executes a simple HTTP request and returns the response."

  # Declares that this runner needs a dependency providing an "http_client" asset.
  # The key "client" maps to the field name in the Go handler's Deps struct.
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

    default = "GET"
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