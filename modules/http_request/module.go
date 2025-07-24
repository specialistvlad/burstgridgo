package http_request

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/vk/burstgridgo/internal/engine"
)

// Input defines the arguments for the http_request runner.
// The `hcl` tags correspond to the `input` block in the manifest.
type Input struct {
	URL    string `hcl:"url"`
	Method string `hcl:"method,optional"`
}

// Output defines the values produced by the http_request runner.
// The `cty` tags are used to expose the fields back to HCL.
type Output struct {
	StatusCode int    `cty:"status_code"`
	Body       string `cty:"body"`
}

// OnRunHttpRequest is the handler for the 'http_request' runner's on_run event.
func OnRunHttpRequest(ctx context.Context, input *Input) (*Output, error) {
	slog.Info("Making HTTP request", "method", input.Method, "url", input.URL)

	// NOTE: We will address the shared http.Client in a later step as per our roadmap.
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

	return &Output{
		StatusCode: resp.StatusCode,
		Body:       string(bodyBytes),
	}, nil
}

// init registers the handler with the engine.
func init() {
	engine.RegisterHandler("OnRunHttpRequest", &engine.RegisteredHandler{
		NewInput: func() any { return new(Input) },
		Fn:       OnRunHttpRequest,
	})
}
