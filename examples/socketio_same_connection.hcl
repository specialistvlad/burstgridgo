## This example demonstrates how to use a single persistent Socket.IO connection
## IT IS NOT IMPLEMENTED IN THE CODE YET

step "env_vars" "env" {}

# 1. Establish one, persistent connection.
step "socketio" "shared_client" {
  arguments {
    url                  = step.env_vars.env.output.all.SOCKETIO_WSS_URL
    insecure_skip_verify = true
    # Note: This runner would need to be modified to output a client object.
  }
}

# 2. Use the shared client to send the first request.
step "socketio_request" "get_upload_info" {
  arguments {
    # This `client` argument does not yet exist on any runner.
    client     = step.socketio.shared_client.output.client
    emit_event = "document.post.v2.request"
    on_event   = "document.post.v2.success"
    timeout    = "5s"
    emit_data  = { file_name = "report-final-v2.pdf" }
  }
}

# 3. Upload the file to S3.
step "s3" "upload" {
  arguments {
    action      = "upload"
    source_path = step.env_vars.env.output.all.UPLOAD_FILE_PATH
    upload_url  = step.socketio_request.get_upload_info.output.response_data.uploading_url
  }
}