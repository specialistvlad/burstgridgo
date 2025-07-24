runner "s3" {
  description = "Performs actions on S3, such as uploading to a pre-signed URL."

  input "action" {
    type        = string
    description = "The action to perform (e.g., 'upload')."
  }

  input "source_path" {
    type        = string
    description = "The local path to the file to upload."
    optional    = true
  }

  input "upload_url" {
    type        = string
    description = "The pre-signed S3 URL to upload the file to."
    optional    = true
  }

  output "success" {
    type        = bool
    description = "True if the operation was successful."
  }

  output "status" {
    type        = string
    description = "The HTTP status of the operation."
  }

  lifecycle {
    on_run = "OnRunS3"
  }
}