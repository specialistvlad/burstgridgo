package env_vars

import (
	"context"
	"log/slog"
	"os"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/vk/burstgridgo/internal/engine"
	"github.com/zclconf/go-cty/cty"
)

type EnvVarsRunner struct{}

func (r *EnvVarsRunner) Run(ctx context.Context, mod engine.Module, evalCtx *hcl.EvalContext) (cty.Value, error) {
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

func init() {
	engine.Registry["env_vars"] = &EnvVarsRunner{}
	slog.Debug("Runner registered", "runner", "env_vars")
}
