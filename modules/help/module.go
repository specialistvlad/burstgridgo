package help

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/hashicorp/hcl/v2"
	"github.com/vk/burstgridgo/internal/engine"
	"github.com/zclconf/go-cty/cty"
)

// HelpRunner implements the engine.Runner interface to display help text.
type HelpRunner struct{}

// Run executes the logic for the help module.
func (r *HelpRunner) Run(ctx context.Context, mod engine.Module, evalCtx *hcl.EvalContext) (cty.Value, error) {
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

Options:
  -g, --grid string
    Explicitly specify the path to the grid file or directory.

  --log-format string
    Log output format. Options: 'text' (default) or 'json'.

  --log-level string
    Set the logging level. Options: 'debug', 'info' (default), 'warn', 'error'.

  --healthcheck-port int
    Port for the HTTP health check server. (default: 8080)

Examples:
  # Run a grid with detailed debug logging
  burstgridgo --log-level=debug ./grids/my_test.hcl

  # Run with targeted verbose logs for the 'login' and 'upload' modules
  BGGO_DEBUG_MODULES=login,upload burstgridgo --log-level=debug ./grids/my_test.hcl
`
	fmt.Println(helpText)

	return cty.NullVal(cty.DynamicPseudoType), nil
}

// init registers the help runner with the engine's registry.
func init() {
	engine.Registry["help"] = &HelpRunner{}
	slog.Debug("Runner registered", "runner", "help")
}
