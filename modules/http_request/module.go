package http_request

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/vk/burstgridgo/internal/engine"
	"github.com/zclconf/go-cty/cty"
)

// Input defines the arguments for the http_request runner.
type Input struct {
	URL    string `hcl:"url"`
	Method string `hcl:"method,optional"`
}

// The native Go output struct. We can keep this for internal clarity.
type Output struct {
	StatusCode int
	Body       string
}

// OnRunHttpRequest is the handler for the 'http_request' runner's on_run event.
func OnRunHttpRequest(ctx context.Context, input *Input) (any, error) {
	slog.Info("Making HTTP request", "method", input.Method, "url", input.URL)

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequestWithContext(ctx, input.Method, input.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	slog.Info("Received HTTP response", "status", resp.Status)

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Manually construct the cty.Value for the output.
	outputObject := cty.ObjectVal(map[string]cty.Value{
		"status_code": cty.NumberIntVal(int64(resp.StatusCode)),
		"body":        cty.StringVal(string(bodyBytes)),
	})

	// Wrap the output in an "output" attribute to match the HCL access pattern.
	return cty.ObjectVal(map[string]cty.Value{
		"output": outputObject,
	}), nil
}

// init registers the handler with the engine.
func init() {
	engine.RegisterHandler("OnRunHttpRequest", &engine.RegisteredHandler{
		NewInput: func() any { return new(Input) },
		Fn:       OnRunHttpRequest,
	})
}
