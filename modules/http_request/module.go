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
	"github.com/zclconf/go-cty/cty"
)

// HTTPRequestRunner implements the engine.Runner interface for HTTP requests.
type HTTPRequestRunner struct{}

// Run executes the logic for an http_request module.
func (r *HTTPRequestRunner) Run(mod engine.Module, ctx *hcl.EvalContext) (cty.Value, error) {
	log.Printf("    ⚙️  Executing http_request runner for module '%s'...", mod.Name)

	// 1. Decode the specific config from the module's body.
	config, err := DecodeConfig(mod.Body, ctx)
	if err != nil {
		return cty.NullVal(cty.DynamicPseudoType), fmt.Errorf("failed to decode config for module %s: %w", mod.Name, err)
	}

	// 2. Prepare and execute the HTTP request.
	method := "GET"
	if config.Method != "" {
		method = strings.ToUpper(config.Method)
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest(method, config.URL, nil)
	if err != nil {
		return cty.NullVal(cty.DynamicPseudoType), fmt.Errorf("failed to create request for module %s: %w", mod.Name, err)
	}

	log.Printf("    ➡️   Making %s request to %s", method, config.URL)
	resp, err := client.Do(req)
	if err != nil {
		return cty.NullVal(cty.DynamicPseudoType), fmt.Errorf("failed to execute request for module %s: %w", mod.Name, err)
	}
	defer resp.Body.Close()

	// 3. Validate the response.
	log.Printf("    ⬅️   Received status: %s (%d)", resp.Status, resp.StatusCode)
	if config.Expect != nil && config.Expect.Status != 0 {
		if resp.StatusCode != config.Expect.Status {
			return cty.NullVal(cty.DynamicPseudoType), fmt.Errorf("unexpected status for module %s: got %d, want %d", mod.Name, resp.StatusCode, config.Expect.Status)
		}
	}

	log.Printf("    ✅ Successfully executed module '%s'.", mod.Name)
	// This module produces no output for other modules.
	return cty.NullVal(cty.DynamicPseudoType), nil
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

func DecodeConfig(body hcl.Body, ctx *hcl.EvalContext) (*Config, error) {
	var config Config
	diags := gohcl.DecodeBody(body, ctx, &config)
	if diags.HasErrors() {
		return nil, diags
	}
	return &config, nil
}
