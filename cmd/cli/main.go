package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/vk/burstgridgo/internal/app"
	"github.com/vk/burstgridgo/internal/cli"
	"github.com/vk/burstgridgo/internal/hcl"
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
			fmt.Fprintln(os.Stderr, exitErr.Message)
			os.Exit(exitErr.Code)
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// run encapsulates the main application logic for easier testing and error handling.
func run(outW io.Writer, args []string) error {
	appConfig, shouldExit, err := cli.Parse(args, outW)
	if err != nil {
		return err
	}
	if shouldExit {
		return nil
	}

	// The app panics on critical config errors, so we recover here to provide
	// a clean exit message to the user.
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(outW, "A critical startup error occurred: %v\n", r)
			os.Exit(1)
		}
	}()

	// Instantiate the concrete HCL loader to pass to the app.
	loader := hcl.NewLoader()
	burstgridApp := app.NewApp(outW, appConfig, loader)

	return burstgridApp.Run(context.Background(), appConfig)
}
