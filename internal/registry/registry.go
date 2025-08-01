package registry

import (
	"reflect"

	"github.com/vk/burstgridgo/internal/config"
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
	DefinitionRegistry      map[string]*config.RunnerDefinition
	AssetDefinitionRegistry map[string]*config.AssetDefinition
	AssetInterfaceRegistry  map[string]reflect.Type
}

// New creates and initializes a new Registry instance.
func New() *Registry {
	return &Registry{
		HandlerRegistry:         make(map[string]*RegisteredRunner),
		AssetHandlerRegistry:    make(map[string]*RegisteredAsset),
		DefinitionRegistry:      make(map[string]*config.RunnerDefinition),
		AssetDefinitionRegistry: make(map[string]*config.AssetDefinition),
		AssetInterfaceRegistry:  make(map[string]reflect.Type),
	}
}

// PopulateDefinitionsFromModel copies the loaded module definitions from the
// config model into the registry for easy access during execution.
func (r *Registry) PopulateDefinitionsFromModel(model *config.Model) {
	for key, val := range model.Runners {
		r.DefinitionRegistry[key] = val
	}
	for key, val := range model.Assets {
		r.AssetDefinitionRegistry[key] = val
	}
}
