package registry

import (
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
	HandlerRegistry         map[string]*RegisteredRunner
	AssetHandlerRegistry    map[string]*RegisteredAsset
	DefinitionRegistry      map[string]*schema.RunnerDefinition
	AssetDefinitionRegistry map[string]*schema.AssetDefinition
	AssetInterfaceRegistry  map[string]reflect.Type
}

// New creates and initializes a new Registry instance.
func New() *Registry {
	return &Registry{
		HandlerRegistry:         make(map[string]*RegisteredRunner),
		AssetHandlerRegistry:    make(map[string]*RegisteredAsset),
		DefinitionRegistry:      make(map[string]*schema.RunnerDefinition),
		AssetDefinitionRegistry: make(map[string]*schema.AssetDefinition),
		AssetInterfaceRegistry:  make(map[string]reflect.Type),
	}
}
