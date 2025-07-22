package help

import (
	"fmt"
	"log/slog"

	"github.com/hashicorp/hcl/v2"
	"github.com/vk/burstgridgo/internal/engine"
	"github.com/zclconf/go-cty/cty"
)

// HelpRunner implements the engine.Runner interface to display help text.
type HelpRunner struct{}

// Run executes the logic for the help module.
func (r *HelpRunner) Run(mod engine.Module, ctx *hcl.EvalContext) (cty.Value, error) {
	helpText := `
BurstGridGo - A declarative, concurrency-first load testing tool.

Description:
  BurstGridGo executes workflows defined in HCL files, known as "grids".
  It builds a dependency graph from your modules and runs them concurrently,
  respecting the dependencies you define.

Usage:
  burstgridgo [options] [GRID_PATH]

Arguments:
  GRID_PATH
    Path to a single .hcl file or a directory containing .hcl files.
    If a directory is specified, it will be scanned recursively.

Options:
  -g, --grid string
    Explicitly specify the path to the grid file or directory.
  
  --log-format string
    Log output format. Options: 'text' (default) or 'json'.
  
  --healthcheck-port int
    Port for the HTTP health check server. Set to 0 to disable. (default: 8080)

Examples:
  # Run all .hcl files in the 'signup_workflow' directory
  burstgridgo ./grids/signup_workflow

  # Run a single grid file with JSON-formatted logs
  burstgridgo --log-format=json ./grids/smoke_test.hcl
`
	fmt.Println(helpText)

	return cty.NullVal(cty.DynamicPseudoType), nil
}

// init registers the help runner with the engine's registry.
func init() {
	engine.Registry["help"] = &HelpRunner{}
	slog.Debug("Runner registered", "runner", "help")
}
