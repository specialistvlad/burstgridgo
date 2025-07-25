# This example demonstrates how a stateful Socket.IO module could be built
# using the new resource management system.
# NOTE: The 'socketio_client' asset and 'socketio_request' runner are
# hypothetical and have not been implemented yet.

step "env_vars" "env" {}

# 1. Define a stateful resource for the persistent Socket.IO connection.
# The engine would call the 'create' handler for this asset once.
resource "socketio_client" "shared_connection" {
  arguments {
    url                  = step.env_vars.env.output.all.SOCKETIO_WSS_URL
    insecure_skip_verify = true
  }
}

# 2. Use the shared client to send the first request.
# The 'uses' block injects the live client object into the step's handler.
step "socketio_request" "get_upload_info" {
  uses {
    client = resource.socketio_client.shared_connection
  }
  arguments {
    emit_event = "document.post.v2.request"
    on_event   = "document.post.v2.success"
    timeout    = "5s"
    emit_data  = { file_name = "report-final-v2.pdf" }
  }
}

# 3. Upload the file to S3 using data from the previous step.
step "s3" "upload" {
  arguments {
    action      = "upload"
    source_path = step.env_vars.env.output.all.UPLOAD_FILE_PATH
    upload_url  = step.socketio_request.get_upload_info.output.response_data.uploading_url
  }
}