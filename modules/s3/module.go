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

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/vk/burstgridgo/internal/engine"
	"github.com/zclconf/go-cty/cty"
)

// S3Runner implements the engine.Runner interface for S3 actions.
type S3Runner struct{}

// Config defines the HCL structure for the S3 module.
type Config struct {
	Action     string `hcl:"action"`
	SourcePath string `hcl:"source_path,optional"`
	UploadURL  string `hcl:"upload_url,optional"`
}

// handleUpload contains the logic for uploading a file to a pre-signed URL.
func handleUpload(ctx context.Context, config *Config, logger *slog.Logger) (cty.Value, error) {
	file, err := os.Open(config.SourcePath)
	if err != nil {
		return cty.NullVal(cty.DynamicPseudoType), fmt.Errorf("failed to open source file '%s': %w", config.SourcePath, err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return cty.NullVal(cty.DynamicPseudoType), fmt.Errorf("failed to get file stats for '%s': %w", config.SourcePath, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, config.UploadURL, file)
	if err != nil {
		return cty.NullVal(cty.DynamicPseudoType), fmt.Errorf("failed to create S3 upload request: %w", err)
	}

	contentType := mime.TypeByExtension(filepath.Ext(config.SourcePath))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	req.Header.Set("Content-Type", contentType)
	req.ContentLength = stat.Size()

	logger.Info("Uploading file to S3", "source", config.SourcePath, "size", stat.Size(), "contentType", contentType)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return cty.NullVal(cty.DynamicPseudoType), fmt.Errorf("failed to execute S3 upload request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return cty.NullVal(cty.DynamicPseudoType), fmt.Errorf("S3 upload failed with status: %s", resp.Status)
	}

	logger.Info("Successfully uploaded file", "status", resp.Status)

	return cty.ObjectVal(map[string]cty.Value{
		"success": cty.BoolVal(true),
		"status":  cty.StringVal(resp.Status),
	}), nil
}

func (r *S3Runner) Run(ctx context.Context, mod engine.Module, evalCtx *hcl.EvalContext) (cty.Value, error) {
	var config Config
	if diags := gohcl.DecodeBody(mod.Body, evalCtx, &config); diags.HasErrors() {
		return cty.NullVal(cty.DynamicPseudoType), diags
	}

	logger := slog.With("module", mod.Name, "action", config.Action)

	switch strings.ToLower(config.Action) {
	case "upload":
		return handleUpload(ctx, &config, logger)
	case "download":
		return cty.NullVal(cty.DynamicPseudoType), fmt.Errorf("s3 action 'download' is not yet implemented")
	default:
		return cty.NullVal(cty.DynamicPseudoType), fmt.Errorf("unknown s3 action: '%s'", config.Action)
	}
}

// init registers the new "s3" runner.
func init() {
	engine.Registry["s3"] = &S3Runner{}
	slog.Debug("Runner registered", "runner", "s3")
}
