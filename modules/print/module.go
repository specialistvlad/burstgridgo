package print

import (
	"context"
	"reflect"

	"github.com/specialistvlad/burstgridgo/internal/ctxlog"
	"github.com/specialistvlad/burstgridgo/internal/handlers"
)

// Input defines the arguments for the print runner. It accepts any single value.
type Input struct {
	Value any `bggo:"input"`
}

type Deps struct{}

// OnRunPrint is the handler for the 'print' runner's on_run lifecycle event.
// It logs the provided input value using the contextual structured logger.
func OnRunPrint(ctx context.Context, deps *Deps, input *Input) (any, error) {
	logger := ctxlog.FromContext(ctx)

	// Log the received value. slog handles formatting for various types
	// (primitives, maps, slices, structs) automatically.
	logger.Info("Printing input value", "value", input.Value)

	return nil, nil
}

// Register registers the handler with the engine.
func RegisterHandler(hndl *handlers.Handlers) {
	hndl.RegisterHandler("OnRunPrint", &handlers.RegisteredHandler{
		Input:     func() any { return new(Input) },
		InputType: reflect.TypeOf(Input{}),
		Deps:      func() any { return new(Deps) },
		Fn:        OnRunPrint,
	})
}

// package registry

// import (
// 	"github.com/specialistvlad/burstgridgo/internal/handlers"
// 	"github.com/specialistvlad/burstgridgo/internal/runner"
// )

// // Module is the interface that all core modules must implement to be registered.
// type Module interface {
// 	Register(r *Registry)
// }

// // Registry holds all the registered handlers, definitions, and interfaces for
// // a single application instance.
// type Registry struct {
// 	handlersRegistry handlers.Handlers
// 	runnersRegistry  []*runner.Runner
// }

// // New creates and initializes a new Registry instance.
// // If a nil handlers object is provided, it creates a new empty one as a fallback.
// func New(h *handlers.Handlers) *Registry {
// 	if h == nil {
// 		h = handlers.New()
// 	}
// 	return &Registry{
// 		handlersRegistry: *h,
// 		runnersRegistry:  []*runner.Runner{},
// 	}
// }
