package env_vars

import (
	"context"
	"os"
	"strings"

	"github.com/vk/burstgridgo/internal/engine"
	"github.com/zclconf/go-cty/cty"
)

func OnRunEnvVars(ctx context.Context, input any) (any, error) {
	envMap := make(map[string]cty.Value)
	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		if len(pair) == 2 {
			envMap[pair[0]] = cty.StringVal(pair[1])
		}
	}

	// This is the data we want to expose as the "output" block.
	outputObject := cty.ObjectVal(map[string]cty.Value{
		"all": cty.MapVal(envMap),
	})

	// CORRECTED: Wrap the output object in another object with a single "output" key.
	// This makes the final data structure match the HCL expression `step.<name>.output.all`.
	return cty.ObjectVal(map[string]cty.Value{
		"output": outputObject,
	}), nil
}

func init() {
	engine.RegisterHandler("OnRunEnvVars", &engine.RegisteredHandler{
		NewInput: nil,
		Fn:       OnRunEnvVars,
	})
}
