package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/vk/burstgridgo/internal/app"
	"github.com/vk/burstgridgo/internal/cli"
	"github.com/vk/burstgridgo/internal/hcl_adapter"
)

// main is the entrypoint for the burstgridgo application.
func main() {
	// Use a minimal logger until the full one is configured.
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	// The real main function handles errors and exit codes.
	if err := run(os.Stdout, os.Args[1:]); err != nil {
		if exitErr, ok := err.(*cli.ExitError); ok {
			// cli.Parse returns specific exit codes for argument errors.
			fmt.Fprintln(os.Stderr, exitErr.Message)
			os.Exit(exitErr.Code)
		}
		// All other errors, including panics from run(), result in a generic exit code 1.
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// run encapsulates the main application logic.
func run(outW io.Writer, args []string) (err error) {
	appConfig, shouldExit, err := cli.Parse(args, outW)
	if err != nil {
		return err
	}
	if shouldExit {
		return nil
	}

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("application startup panicked: %v", r)
		}
	}()

	// Instantiate the concrete HCL loader to pass to the app.
	loader := hcl_adapter.NewLoader()
	burstgridApp := app.NewApp(outW, appConfig, loader)

	return burstgridApp.Run(context.Background(), appConfig)
}
