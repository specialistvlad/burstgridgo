package http_client

import (
	"context"
	"net/http"
	"reflect"
	"time"

	"github.com/vk/burstgridgo/internal/engine"
)

// Input defines the arguments for creating an http_client resource.
type Input struct {
	Timeout string `hcl:"timeout,optional"`
}

// CreateHttpClient is the 'create' handler for the asset.
// It returns a live *http.Client object that will be shared.
func CreateHttpClient(ctx context.Context, input *Input) (*http.Client, error) {
	timeout, err := time.ParseDuration(input.Timeout)
	if err != nil {
		return nil, err
	}

	client := &http.Client{
		Timeout: timeout,
		// In a real-world scenario, you would configure the transport here
		// for connection pooling, etc.
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		},
	}
	return client, nil
}

// DestroyHttpClient is the 'destroy' handler. For an http.Client,
// we just need to close idle connections.
func DestroyHttpClient(client *http.Client) error {
	client.CloseIdleConnections()
	return nil
}

func init() {
	// Register the asset's Go handlers with the engine.
	engine.RegisterAssetHandler("CreateHttpClient", &engine.RegisteredAssetHandler{
		NewInput: func() any { return new(Input) },
		CreateFn: CreateHttpClient,
	})
	engine.RegisterAssetHandler("DestroyHttpClient", &engine.RegisteredAssetHandler{
		DestroyFn: DestroyHttpClient,
	})

	// Register the Go interface that this asset provides.
	// This enables compile-time-like validation in the executor.
	engine.RegisterAssetInterface("http_client", reflect.TypeOf((*http.Client)(nil)))
}
