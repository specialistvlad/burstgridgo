package print

import (
	"context"
	"reflect"

	"github.com/vk/burstgridgo/internal/ctxlog"
	"github.com/vk/burstgridgo/internal/registry"
)

// Module implements the registry.Module interface for this package.
type Module struct{}

// Input defines the arguments for the print runner. It accepts any single value.
type Input struct {
	Value any `bggo:"input"`
}

// Deps is an empty struct because this runner does not use any resources.
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
func (m *Module) Register(r *registry.Registry) {
	r.RegisterRunner("OnRunPrint", &registry.RegisteredRunner{
		NewInput:  func() any { return new(Input) },
		InputType: reflect.TypeOf(Input{}),
		NewDeps:   func() any { return new(Deps) },
		Fn:        OnRunPrint,
	})
}
