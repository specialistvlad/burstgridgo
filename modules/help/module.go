package help

import (
	"context"
	"fmt"

	"github.com/vk/burstgridgo/internal/engine"
	"github.com/zclconf/go-cty/cty"
)

// Deps is an empty struct because this runner does not use any resources.
type Deps struct{}

// OnRunHelp is the handler for the 'help' runner's on_run lifecycle event.
func OnRunHelp(ctx context.Context, deps *Deps, input any) (cty.Value, error) {
	helpText := `
BurstGridGo - A declarative, concurrency-first load testing tool.

Description:
  BurstGridGo executes workflows defined in HCL files, known as "grids".
  It builds a dependency graph from them and runs them concurrently,
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
    Port for the HTTP health check server. 0 to disabled. (default: 0)
`
	fmt.Println(helpText)
	return cty.NilVal, nil
}

// init registers the handler with the engine.
func init() {
	engine.RegisterHandler("OnRunHelp", &engine.RegisteredHandler{
		NewInput: func() any { return nil }, // No 'arguments' block.
		NewDeps:  func() any { return new(Deps) },
		Fn:       OnRunHelp,
	})
}
