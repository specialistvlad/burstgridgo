runner "http_request" {
  description = "Executes a simple HTTP request and returns the response."

  input "url" {
    type        = string
    description = "The URL to send the request to."
  }

  input "method" {
    type        = string
    description = "The HTTP method to use."
    optional    = true
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