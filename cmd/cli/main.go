// Command burstgridgo is a declarative, concurrency-first load testing tool.
// It executes workflows defined in HCL files by building a dependency graph
// and running the steps concurrently, respecting all defined dependencies.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/vk/burstgridgo/internal/app"
	"github.com/vk/burstgridgo/internal/cli"
)

// run is the primary, testable entrypoint for the application. It orchestrates
// the parsing of CLI arguments and the execution of the main application logic.
// It returns a non-nil error only if the application itself fails fatally.
func run(ctx context.Context, args []string, outW io.Writer) error {
	appConfig, shouldExit, err := cli.Parse(args, outW)
	if err != nil {
		return err // Pass the ExitError up to main
	}
	if shouldExit {
		return nil
	}

	// The main application call uses the default core modules.
	a := app.NewApp(outW, appConfig)

	if err := a.Run(ctx, appConfig); err != nil {
		// Wrap application errors in a standard ExitError.
		return &cli.ExitError{Code: 1, Message: err.Error()}
	}
	return nil
}

// main is the ultimate entrypoint for the executable. It wraps the run
// function to handle process-level concerns like exit codes.
func main() {
	ctx := context.Background()
	if err := run(ctx, os.Args[1:], os.Stdout); err != nil {
		var exitErr *cli.ExitError
		if errors.As(err, &exitErr) {
			if exitErr.Message != "" {
				fmt.Fprintln(os.Stderr, exitErr.Message)
			}
			os.Exit(exitErr.Code)
		}
		fmt.Fprintf(os.Stderr, "an unexpected error occurred: %v\n", err)
		os.Exit(1)
	}
}
