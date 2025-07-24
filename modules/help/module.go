package help

import (
	"context"
	"fmt"

	"github.com/vk/burstgridgo/internal/engine"
)

// OnRunHelp is the handler for the 'help' runner's on_run lifecycle event.
// It matches the standard signature func(ctx, input) (output, error).
func OnRunHelp(ctx context.Context, input any) (any, error) {
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
    Port for the HTTP health check server. (default: 8080)
`
	fmt.Println(helpText)
	return nil, nil
}

// init registers the handler with the engine.
func init() {
	engine.RegisterHandler("OnRunHelp", &engine.RegisteredHandler{
		NewInput: nil, // This handler takes no input.
		Fn:       OnRunHelp,
	})
}
