package env_vars

import (
	"context"
	"os"
	"strings"

	"github.com/vk/burstgridgo/internal/engine"
)

// Output defines the values produced by the env_vars runner.
type Output struct {
	All map[string]string `cty:"all"`
}

// OnRunEnvVars is the handler for the 'env_vars' runner's on_run lifecycle event.
func OnRunEnvVars(ctx context.Context) (*Output, error) {
	envMap := make(map[string]string)
	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		if len(pair) == 2 {
			envMap[pair[0]] = pair[1]
		}
	}

	return &Output{All: envMap}, nil
}

// init registers the handler with the engine.
func init() {
	engine.RegisterHandler("OnRunEnvVars", &engine.RegisteredHandler{
		NewInput: nil, // This runner takes no input.
		Fn:       OnRunEnvVars,
	})
}
