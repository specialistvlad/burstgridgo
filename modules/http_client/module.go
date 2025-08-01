// Package http_client provides a stateful, shareable HTTP client asset and a
// stateless runner for making individual HTTP requests.
package http_client

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"reflect"
	"time"

	"github.com/vk/burstgridgo/internal/registry"
)

// Module implements the registry.Module interface.
type Module struct{}

// Register registers all of the module's components (assets and runners).
func (m *Module) Register(r *registry.Registry) {
	// Register the stateful http_client asset.
	r.RegisterAssetHandler("CreateHttpClient", &registry.RegisteredAsset{
		NewInput: func() any { return new(AssetInput) },
		CreateFn: createHttpClient,
	})
	r.RegisterAssetHandler("DestroyHttpClient", &registry.RegisteredAsset{
		DestroyFn: destroyHttpClient,
	})
	r.RegisterAssetInterface("http_client", reflect.TypeOf((*http.Client)(nil)))

	// Register the stateless http_request runner.
	r.RegisterRunner("OnRunHttpRequest", &registry.RegisteredRunner{
		NewInput:  func() any { return new(RunnerInput) },
		InputType: reflect.TypeOf(RunnerInput{}),
		NewDeps:   func() any { return new(RunnerDeps) },
		Fn:        onRunHttpRequest,
	})
}

// --- Asset: http_client ---

// AssetInput defines the arguments for creating an http_client resource.
type AssetInput struct {
	Timeout string `bggo:"timeout"`
}

// createHttpClient is the 'create' handler for the asset.
func createHttpClient(ctx context.Context, input *AssetInput) (*http.Client, error) {
	timeout, err := time.ParseDuration(input.Timeout)
	if err != nil {
		return nil, err
	}
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		},
	}, nil
}

// destroyHttpClient is the 'destroy' handler for the asset.
func destroyHttpClient(client *http.Client) error {
	client.CloseIdleConnections()
	return nil
}

// --- Runner: http_request ---

// RunnerInput defines the arguments for the 'arguments' HCL block of the runner.
type RunnerInput struct {
	URL    string `bggo:"url"`
	Method string `bggo:"method"`
}

// RunnerDeps defines the injected resources from the 'uses' HCL block.
type RunnerDeps struct {
	Client *http.Client `bggo:"client"`
}

// RunnerOutput defines the data structure returned by the runner.
type RunnerOutput struct {
	StatusCode int    `cty:"status_code"`
	Body       string `cty:"body"`
}

// onRunHttpRequest is the handler for the 'http_request' runner's on_run event.
func onRunHttpRequest(ctx context.Context, deps *RunnerDeps, input *RunnerInput) (*RunnerOutput, error) {
	slog.Info("Making HTTP request", "method", input.Method, "url", input.URL)

	if deps.Client == nil {
		return nil, fmt.Errorf("http client dependency was not injected")
	}

	req, err := http.NewRequestWithContext(ctx, input.Method, input.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := deps.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	slog.Info("Received HTTP response", "status", resp.Status)

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return &RunnerOutput{
		StatusCode: resp.StatusCode,
		Body:       string(bodyBytes),
	}, nil
}
