package http_request

import (
	"fmt"
	"log"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/vk/burstgridgo/internal/engine" // Import the engine package
)

// HTTPRequestRunner implements the engine.Runner interface for HTTP requests.
type HTTPRequestRunner struct{}

// Run executes the logic for an http_request module.
func (r *HTTPRequestRunner) Run(mod engine.Module) error {
	log.Printf("    ⚙️ Executing http_request runner for module '%s'...", mod.Name)

	// 1. Decode the specific config from the module's body.
	config, err := DecodeConfig(mod.Body)
	if err != nil {
		return fmt.Errorf("failed to decode config for module %s: %w", mod.Name, err)
	}

	// 2. (Placeholder) Execute the actual HTTP request logic here.
	log.Printf("    ➡️  Making %s request to %s", config.Method, config.URL)
	log.Printf("    ➡️  Expecting status: %d", config.Expect.Status)
	// httpClient := &http.Client{}
	// resp, err := httpClient.Get(config.URL) ...

	log.Printf("    ✅ Successfully executed module '%s'.", mod.Name)
	return nil
}

// init registers the http_request runner with the engine's registry.
func init() {
	// The key "http-request" must match the 'runner' attribute in your HCL files.
	engine.Registry["http-request"] = &HTTPRequestRunner{}
	log.Println("🔌 http-request runner registered.")
}

// --- Specific Config Structs and Decoder ---

// HTTPExpect defines the expected HTTP response properties for a request.
type HTTPExpect struct {
	Status int `hcl:"status,optional"`
}

// Config defines the specific configuration for an http_request module.
type Config struct {
	Method string      `hcl:"method,optional"`
	URL    string      `hcl:"url"` // The URL is required for this module
	Expect *HTTPExpect `hcl:"expect,block"`
}

// DecodeConfig decodes the body of an http_request module block into its specific Config struct.
func DecodeConfig(body hcl.Body) (*Config, error) {
	var config Config
	diags := gohcl.DecodeBody(body, nil, &config)
	if diags.HasErrors() {
		return nil, diags
	}
	return &config, nil
}
