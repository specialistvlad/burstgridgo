package http_request

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/vk/burstgridgo/internal/engine"
)

// HTTPRequestRunner implements the engine.Runner interface for HTTP requests.
type HTTPRequestRunner struct{}

// Run executes the logic for an http_request module.
func (r *HTTPRequestRunner) Run(mod engine.Module) error {
	log.Printf("    ⚙️  Executing http_request runner for module '%s'...", mod.Name)

	// 1. Decode the specific config from the module's body.
	config, err := DecodeConfig(mod.Body)
	if err != nil {
		return fmt.Errorf("failed to decode config for module %s: %w", mod.Name, err)
	}

	// 2. Prepare and execute the HTTP request.
	// Default to GET if no method is specified.
	method := "GET"
	if config.Method != "" {
		method = strings.ToUpper(config.Method)
	}

	// Create a client with a timeout.
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Create the request.
	req, err := http.NewRequest(method, config.URL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request for module %s: %w", mod.Name, err)
	}
	// TODO: Add headers from config here if needed, e.g., req.Header.Add("Content-Type", "application/json")

	log.Printf("    ➡️   Making %s request to %s", method, config.URL)
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request for module %s: %w", mod.Name, err)
	}
	defer resp.Body.Close()

	// 3. Validate the response.
	log.Printf("    ⬅️   Received status: %s (%d)", resp.Status, resp.StatusCode)
	if config.Expect != nil && config.Expect.Status != 0 {
		if resp.StatusCode != config.Expect.Status {
			return fmt.Errorf("unexpected status for module %s: got %d, want %d", mod.Name, resp.StatusCode, config.Expect.Status)
		}
	}

	log.Printf("    ✅ Successfully executed module '%s'.", mod.Name)
	return nil
}

// init registers the http_request runner with the engine's registry.
func init() {
	engine.Registry["http-request"] = &HTTPRequestRunner{}
	log.Println("🔌 http-request runner registered.")
}

// --- Specific Config Structs and Decoder ---

type HTTPExpect struct {
	Status int `hcl:"status,optional"`
}

type Config struct {
	Method string      `hcl:"method,optional"`
	URL    string      `hcl:"url"`
	Expect *HTTPExpect `hcl:"expect,block"`
}

func DecodeConfig(body hcl.Body) (*Config, error) {
	var config Config
	diags := gohcl.DecodeBody(body, nil, &config)
	if diags.HasErrors() {
		return nil, diags
	}
	return &config, nil
}
