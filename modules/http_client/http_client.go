// Package http_client provides a stateful, shareable HTTP client asset and a
// stateless runner for making individual HTTP requests.
package http_client

import (
	"net/http"
	"reflect"

	"github.com/vk/burstgridgo/internal/registry"
)

// Module implements the registry.Module interface. It's the main entrypoint
// for the http_client module, responsible for registering all of its
// components with the application's registry.
type Module struct{}

// Register registers all of the module's components (assets and runners) with
// the central registry.
func (m *Module) Register(r *registry.Registry) {
	// Register the stateful http_client asset and its lifecycle handlers.
	r.RegisterAssetHandler("CreateHttpClient", &registry.RegisteredAsset{
		NewInput: func() any { return new(AssetInput) },
		CreateFn: createHttpClient,
	})
	r.RegisterAssetHandler("DestroyHttpClient", &registry.RegisteredAsset{
		DestroyFn: destroyHttpClient,
	})
	r.RegisterAssetInterface("http_client", reflect.TypeOf((*http.Client)(nil)))

	// Register the stateless http_request runner and its handler.
	r.RegisterRunner("OnRunHttpRequest", &registry.RegisteredRunner{
		NewInput: func() any { return new(RunnerInput) },
		NewDeps:  func() any { return new(RunnerDeps) },
		Fn:       onRunHttpRequest,
	})
}
