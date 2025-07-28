package registry

import (
	"fmt"
	"log/slog"
	"reflect"

	"github.com/vk/burstgridgo/internal/schema"
)

// Module is the interface that all core modules must implement to be registered.
type Module interface {
	Register(r *Registry)
}

// Registry holds all the registered handlers, definitions, and interfaces for
// a single application instance.
type Registry struct {
	HandlerRegistry         map[string]*RegisteredHandler
	AssetHandlerRegistry    map[string]*RegisteredAssetHandler
	DefinitionRegistry      map[string]*schema.RunnerDefinition
	AssetDefinitionRegistry map[string]*schema.AssetDefinition
	AssetInterfaceRegistry  map[string]reflect.Type
}

// New creates and initializes a new Registry instance.
func New() *Registry {
	return &Registry{
		HandlerRegistry:         make(map[string]*RegisteredHandler),
		AssetHandlerRegistry:    make(map[string]*RegisteredAssetHandler),
		DefinitionRegistry:      make(map[string]*schema.RunnerDefinition),
		AssetDefinitionRegistry: make(map[string]*schema.AssetDefinition),
		AssetInterfaceRegistry:  make(map[string]reflect.Type),
	}
}

// RegisteredHandler holds the compiled Go parts of a runner's lifecycle function.
type RegisteredHandler struct {
	NewInput func() any
	NewDeps  func() any
	Fn       any
}

// RegisterHandler registers a Go function for a runner's lifecycle event.
func (r *Registry) RegisterHandler(name string, handler *RegisteredHandler) {
	if _, exists := r.HandlerRegistry[name]; exists {
		panic(fmt.Sprintf("runner handler with name '%s' already registered", name))
	}
	slog.Debug("Registering runner handler.", "name", name)
	r.HandlerRegistry[name] = handler
}

// RegisteredAssetHandler holds Go functions for an asset's lifecycle.
type RegisteredAssetHandler struct {
	NewInput  func() any
	CreateFn  any
	DestroyFn any
}

// RegisterAssetHandler registers Go functions for an asset's lifecycle events.
func (r *Registry) RegisterAssetHandler(name string, handler *RegisteredAssetHandler) {
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
