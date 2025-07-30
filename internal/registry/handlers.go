package registry

import (
	"fmt"
	"log/slog"
	"reflect"
)

// RegisteredRunner holds the compiled Go parts of a runner's lifecycle function.
type RegisteredRunner struct {
	NewInput func() any
	NewDeps  func() any
	Fn       any
}

// RegisterRunner registers a Go function for a runner's lifecycle event.
func (r *Registry) RegisterRunner(name string, handler *RegisteredRunner) {
	if _, exists := r.HandlerRegistry[name]; exists {
		panic(fmt.Sprintf("runner handler with name '%s' already registered", name))
	}
	slog.Debug("Registering runner handler.", "name", name)
	r.HandlerRegistry[name] = handler
}

// RegisteredAsset holds Go functions for an asset's lifecycle.
type RegisteredAsset struct {
	NewInput  func() any
	CreateFn  any
	DestroyFn any
}

// RegisterAssetHandler registers Go functions for an asset's lifecycle events.
func (r *Registry) RegisterAssetHandler(name string, handler *RegisteredAsset) {
	if _, exists := r.AssetHandlerRegistry[name]; exists {
		panic(fmt.Sprintf("asset handler with name '%s' already registered", name))
	}
	slog.Debug("Registering asset handler.", "name", name)
	r.AssetHandlerRegistry[name] = handler
}

// RegisterAssetInterface registers the Go interface contract for an asset type.
func (r *Registry) RegisterAssetInterface(assetType string, iface reflect.Type) {
	if _, exists := r.AssetInterfaceRegistry[assetType]; exists {
		panic(fmt.Sprintf("interface for asset type '%s' already registered", assetType))
	}
	slog.Debug("Registering asset interface.", "assetType", assetType, "interface", iface.String())
	r.AssetInterfaceRegistry[assetType] = iface
}
