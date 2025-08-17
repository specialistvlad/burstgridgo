package testutil

import (
	"context"
	"reflect"

	"github.com/specialistvlad/burstgridgo/internal/handlers"
)

// NoOpModule is a helper that satisfies the TestModule interface and
// registers a single "NoOp" runner. It's useful for tests that should
// fail before execution begins but still need valid HCL that can pass
// registry validation.
type NoOpModule struct{}

// Register registers a single "NoOp" runner that takes no inputs,
// requires no dependencies, and does nothing.
func (m *NoOpModule) Register(r *handlers.Handlers) {
	r.RegisterRunner("NoOp", &handlers.RegisteredHandler{
		Input:     func() any { return new(struct{}) },
		InputType: reflect.TypeOf(struct{}{}),
		Deps:      func() any { return new(struct{}) },
		Fn: func(ctx context.Context, deps any, input any) (any, error) {
			// No operation
			return nil, nil
		},
	})
}
