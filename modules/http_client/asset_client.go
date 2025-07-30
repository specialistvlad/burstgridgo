package http_client

import (
	"context"
	"net/http"
	"time"
)

// AssetInput defines the arguments for creating an http_client resource.
type AssetInput struct {
	Timeout string `hcl:"timeout,optional"`
}

// createHttpClient is the 'create' handler for the asset. It returns a live
// *http.Client object that will be shared across steps.
func createHttpClient(ctx context.Context, input *AssetInput) (*http.Client, error) {
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

// destroyHttpClient is the 'destroy' handler for the asset. For an http.Client,
// we just need to gracefully close any idle connections.
func destroyHttpClient(client *http.Client) error {
	client.CloseIdleConnections()
	return nil
}
