package s3

import (
	"context"
	"fmt"
	"log/slog"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/vk/burstgridgo/internal/engine"
	"github.com/zclconf/go-cty/cty"
)

// httpClient is a shared client for all S3 runner executions to reuse TCP connections.
var httpClient = &http.Client{}

// Input defines the arguments for the s3 runner.
type Input struct {
	Action     string `hcl:"action"`
	SourcePath string `hcl:"source_path,optional"`
	UploadURL  string `hcl:"upload_url,optional"`
}

// handleUpload contains the logic for uploading a file to a pre-signed URL.
// It now returns (any, error) to be compatible with the handler signature.
func handleUpload(ctx context.Context, input *Input) (any, error) {
	logger := slog.With("action", "upload")

	file, err := os.Open(input.SourcePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open source file '%s': %w", input.SourcePath, err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get file stats for '%s': %w", input.SourcePath, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, input.UploadURL, file)
	if err != nil {
		return nil, fmt.Errorf("failed to create S3 upload request: %w", err)
	}

	contentType := mime.TypeByExtension(filepath.Ext(input.SourcePath))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	req.Header.Set("Content-Type", contentType)
	req.ContentLength = stat.Size()

	logger.Info("Uploading file to S3", "source", input.SourcePath, "size", stat.Size(), "contentType", contentType)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute S3 upload request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("S3 upload failed with status: %s", resp.Status)
	}

	logger.Info("Successfully uploaded file", "status", resp.Status)

	// Return a cty.Value object directly with the runner's output.
	return cty.ObjectVal(map[string]cty.Value{
		"success": cty.BoolVal(true),
		"status":  cty.StringVal(resp.Status),
	}), nil
}

// OnRunS3 is the handler for the 's3' runner's on_run lifecycle event.
func OnRunS3(ctx context.Context, input *Input) (any, error) {
	switch strings.ToLower(input.Action) {
	case "upload":
		return handleUpload(ctx, input)
	case "download":
		return nil, fmt.Errorf("s3 action 'download' is not yet implemented")
	default:
		return nil, fmt.Errorf("unknown s3 action: '%s'", input.Action)
	}
}

// init registers the handler with the engine.
func init() {
	engine.RegisterHandler("OnRunS3", &engine.RegisteredHandler{
		NewInput: func() any { return new(Input) },
		Fn:       OnRunS3,
	})
}
