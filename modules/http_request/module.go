package http_request

import (
	"context"
	"fmt"
	"log/slog"
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
func (r *HTTPRequestRunner) Run(ctx context.Context, mod engine.Module, evalCtx *hcl.EvalContext) (cty.Value, error) {
	config, err := DecodeConfig(mod.Body, evalCtx)
	if err != nil {
		return cty.NullVal(cty.DynamicPseudoType), fmt.Errorf("failed to decode config: %w", err)
	}

	method := "GET"
	if config.Method != "" {
		method = strings.ToUpper(config.Method)
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequestWithContext(ctx, method, config.URL, nil)
	if err != nil {
		return cty.NullVal(cty.DynamicPseudoType), fmt.Errorf("failed to create request: %w", err)
	}

	slog.Info("Making HTTP request", "module", mod.Name, "method", method, "url", config.URL)
	resp, err := client.Do(req)
	if err != nil {
		return cty.NullVal(cty.DynamicPseudoType), fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	slog.Info("Received HTTP response", "module", mod.Name, "status", resp.Status)

	if config.Expect != nil && config.Expect.Status != 0 {
		if resp.StatusCode != config.Expect.Status {
			return cty.NullVal(cty.DynamicPseudoType), fmt.Errorf("unexpected status: got %d, want %d", resp.StatusCode, config.Expect.Status)
		}
	}

	// This module produces no output for other modules.
	return cty.NullVal(cty.DynamicPseudoType), nil
}

// init registers the http_request runner with the engine's registry.
func init() {
	engine.Registry["http-request"] = &HTTPRequestRunner{}
	slog.Debug("Runner registered", "runner", "http-request")
}

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
