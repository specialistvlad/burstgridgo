package registry

import (
	"github.com/specialistvlad/burstgridgo/internal/handlers"
	"github.com/specialistvlad/burstgridgo/internal/model"
)

// Module is the interface that all core modules must implement to be registered.
type Module interface {
	Register(r *Registry)
}

// Registry holds all the registered handlers, definitions, and interfaces for
// a single application instance.
type Registry struct {
	handlersRegistry handlers.Handlers
	runnersRegistry  []*model.Runner
}

// New creates and initializes a new Registry instance.
func New(hndl *handlers.Handlers) *Registry {
	if hndl == nil {
		hndl = handlers.New()
	}

	return &Registry{
		handlersRegistry: *hndl,
		runnersRegistry:  []*model.Runner{},
	}
}

// Runners returns the slice of loaded runner definitions. For testing.
func (r *Registry) Runners() []*model.Runner {
	return r.runnersRegistry
}
