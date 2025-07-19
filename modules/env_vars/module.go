package env_vars

import (
	"log"
	"os"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/vk/burstgridgo/internal/engine"
	"github.com/zclconf/go-cty/cty"
)

type EnvVarsRunner struct{}

func (r *EnvVarsRunner) Run(mod engine.Module, ctx *hcl.EvalContext) (cty.Value, error) {
	log.Printf("    ⚙️  Executing env_vars runner for module '%s'...", mod.Name)

	envMap := make(map[string]cty.Value)
	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		if len(pair) == 2 {
			envMap[pair[0]] = cty.StringVal(pair[1])
		}
	}

	// The output is an object with a single attribute "all",
	// which contains the map of all environment variables.
	// This matches the HCL expression "module.all_env_vars.all".
	return cty.ObjectVal(map[string]cty.Value{
		"all": cty.MapVal(envMap),
	}), nil
}

func init() {
	engine.Registry["env_vars"] = &EnvVarsRunner{}
	log.Println("🔌 env_vars runner registered.")
}
