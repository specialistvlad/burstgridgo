package env_vars

import (
	"context"
	"os"
	"strings"

	"github.com/vk/burstgridgo/internal/registry"
	"github.com/zclconf/go-cty/cty"
)

// Module implements the registry.Module interface for this package.
type Module struct{}

// Deps is an empty struct because this runner does not use any resources.
type Deps struct{}

// OnRunEnvVars is the handler for the 'env_vars' runner.
func OnRunEnvVars(ctx context.Context, deps *Deps, input any) (cty.Value, error) {
	envMap := make(map[string]cty.Value)
	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		if len(pair) == 2 {
			envMap[pair[0]] = cty.StringVal(pair[1])
		}
	}

	return cty.ObjectVal(map[string]cty.Value{
		"all": cty.MapVal(envMap),
	}), nil
}

// Register registers the handler with the engine.
func (m *Module) Register(r *registry.Registry) {
	r.RegisterHandler("OnRunEnvVars", &registry.RegisteredHandler{
		NewInput: func() any { return nil }, // No 'arguments' block.
		NewDeps:  func() any { return new(Deps) },
		Fn:       OnRunEnvVars,
	})
}
