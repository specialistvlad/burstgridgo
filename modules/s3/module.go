package s3

import (
	"fmt"
	"log"
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
// Different actions will use different optional attributes.
type Config struct {
	Action     string `hcl:"action"`
	SourcePath string `hcl:"source_path,optional"`
	UploadURL  string `hcl:"upload_url,optional"`
	// Future actions can add their own optional fields here,
	// e.g., DestinationPath, DownloadURL, etc.
}

// handleUpload contains the logic for uploading a file to a pre-signed URL.
func handleUpload(config *Config) (cty.Value, error) {
	// 1. Open the local file.
	file, err := os.Open(config.SourcePath)
	if err != nil {
		return cty.NullVal(cty.DynamicPseudoType), fmt.Errorf("failed to open source file '%s': %w", config.SourcePath, err)
	}
	defer file.Close()

	// 2. Get file stats to determine size.
	stat, err := file.Stat()
	if err != nil {
		return cty.NullVal(cty.DynamicPseudoType), fmt.Errorf("failed to get file stats for '%s': %w", config.SourcePath, err)
	}

	// 3. Create the PUT request.
	req, err := http.NewRequest(http.MethodPut, config.UploadURL, file)
	if err != nil {
		return cty.NullVal(cty.DynamicPseudoType), fmt.Errorf("failed to create S3 upload request: %w", err)
	}

	// 4. Set required headers.
	req.ContentLength = stat.Size()
	contentType := mime.TypeByExtension(filepath.Ext(config.SourcePath))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	req.Header.Set("Content-Type", contentType)

	log.Printf("    ➡️  Uploading '%s' (%d bytes) to S3...", config.SourcePath, stat.Size())

	// 5. Execute the request.
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return cty.NullVal(cty.DynamicPseudoType), fmt.Errorf("failed to execute S3 upload request: %w", err)
	}
	defer resp.Body.Close()

	// 6. Check for a successful response.
	if resp.StatusCode != http.StatusOK {
		return cty.NullVal(cty.DynamicPseudoType), fmt.Errorf("S3 upload failed with status: %s", resp.Status)
	}

	log.Printf("    ✅ Successfully uploaded file.")

	return cty.ObjectVal(map[string]cty.Value{
		"success": cty.BoolVal(true),
		"status":  cty.StringVal(resp.Status),
	}), nil
}

func (r *S3Runner) Run(mod engine.Module, ctx *hcl.EvalContext) (cty.Value, error) {
	log.Printf("    ⚙️  Executing s3 runner for module '%s'...", mod.Name)

	var config Config
	if diags := gohcl.DecodeBody(mod.Body, ctx, &config); diags.HasErrors() {
		return cty.NullVal(cty.DynamicPseudoType), diags
	}

	// Use a switch to delegate to the correct handler based on the action.
	switch strings.ToLower(config.Action) {
	case "upload":
		return handleUpload(&config)
	case "download":
		return cty.NullVal(cty.DynamicPseudoType), fmt.Errorf("s3 action 'download' is not yet implemented")
	default:
		return cty.NullVal(cty.DynamicPseudoType), fmt.Errorf("unknown s3 action: '%s'", config.Action)
	}
}

// init registers the new "s3" runner.
func init() {
	engine.Registry["s3"] = &S3Runner{}
	log.Println("🔌 s3 runner registered.")
}
