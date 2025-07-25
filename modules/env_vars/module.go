package env_vars

import (
	"context"
	"os"
	"strings"

	"github.com/vk/burstgridgo/internal/engine"
	"github.com/zclconf/go-cty/cty"
)

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

// init registers the handler with the engine.
func init() {
	engine.RegisterHandler("OnRunEnvVars", &engine.RegisteredHandler{
		NewInput: func() any { return nil }, // No 'arguments' block.
		NewDeps:  func() any { return new(Deps) },
		Fn:       OnRunEnvVars,
	})
}
