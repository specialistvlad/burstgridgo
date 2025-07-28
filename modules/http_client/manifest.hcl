asset "http_client" {
  description = "Provides a shared, persistent HTTP client for connection reuse."

  input "timeout" {
    type        = string
    description = "Request timeout duration (e.g., '30s')."
    optional    = true
    default     = "5s"
  }

  lifecycle {
    create  = "CreateHttpClient"
    destroy = "DestroyHttpClient"
  }
}
