package http_client

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/vk/burstgridgo/internal/registry"
	"github.com/zclconf/go-cty/cty"
)

// HttpRequestModule implements the registry.HttpRequestModule interface for this package.
type HttpRequestModule struct{}

// HttpRequestInput defines the arguments for the 'arguments' HCL block.
type HttpRequestInput struct {
	URL    string `hcl:"url"`
	Method string `hcl:"method,optional"`
}

// HttpRequestDeps defines the injected resources from the 'uses' HCL block.
// The `hcl` tag on the 'Client' field must match the key in the 'uses' block of the manifest.
type HttpRequestDeps struct {
	Client *http.Client `hcl:"client"`
}

// OnRunHttpRequest is the handler for the 'http_request' runner's on_run event.
func OnRunHttpRequest(ctx context.Context, deps *HttpRequestDeps, input *HttpRequestInput) (cty.Value, error) {
	slog.Info("Making HTTP request", "method", input.Method, "url", input.URL)

	if deps.Client == nil {
		return cty.NilVal, fmt.Errorf("http client dependency was not injected")
	}

	req, err := http.NewRequestWithContext(ctx, input.Method, input.URL, nil)
	if err != nil {
		return cty.NilVal, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := deps.Client.Do(req)
	if err != nil {
		return cty.NilVal, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	slog.Info("Received HTTP response", "status", resp.Status)

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return cty.NilVal, fmt.Errorf("failed to read response body: %w", err)
	}

	return cty.ObjectVal(map[string]cty.Value{
		"status_code": cty.NumberIntVal(int64(resp.StatusCode)),
		"body":        cty.StringVal(string(bodyBytes)),
	}), nil
}

// Register registers the handler with the engine.
func (m *HttpRequestModule) Register(r *registry.Registry) {
	r.RegisterRunner("OnRunHttpRequest", &registry.RegisteredRunner{
		NewInput: func() any { return new(HttpRequestInput) },
		NewDeps:  func() any { return new(HttpRequestDeps) },
		Fn:       OnRunHttpRequest,
	})
}
