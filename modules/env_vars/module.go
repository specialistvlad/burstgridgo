package env_vars

import (
	"context"
	"os"
	"reflect"
	"strings"

	"github.com/vk/burstgridgo/internal/registry"
)

// Module implements the registry.Module interface for this package.
type Module struct{}

// Deps is an empty struct because this runner does not use any resources.
type Deps struct{}

// Output defines the data structure returned by the runner.
type Output struct {
	All map[string]string `cty:"all"`
}

// OnRunEnvVars is the handler for the 'env_vars' runner.
func OnRunEnvVars(ctx context.Context, deps *Deps, input any) (*Output, error) {
	envMap := make(map[string]string)
	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		if len(pair) == 2 {
			envMap[pair[0]] = pair[1]
		}
	}

	return &Output{All: envMap}, nil
}

// Register registers the handler with the engine.
func (m *Module) Register(r *registry.Registry) {
	r.RegisterRunner("OnRunEnvVars", &registry.RegisteredRunner{
		NewInput:  func() any { return new(struct{}) }, // No 'arguments' block.
		InputType: reflect.TypeOf(struct{}{}),
		NewDeps:   func() any { return new(Deps) },
		Fn:        OnRunEnvVars,
	})
}
