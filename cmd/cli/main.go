package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/specialistvlad/burstgridgo/internal/app"
	"github.com/specialistvlad/burstgridgo/internal/cli"
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

// run encapsulates the main application logic with robust panic recovery.
func run(outW io.Writer, args []string) (err error) {
	// Defer the panic handler. It will execute when the function exits,
	// either normally or during a panic. If a panic occurred, it recovers
	// and sets the named return variable 'err'.
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("application startup panicked | %v", r)
		}
	}()

	// Use a different variable for the parsing error to avoid shadowing 'err'.
	appConfig, shouldExit, parseErr := cli.Parse(args, outW)
	if parseErr != nil {
		return parseErr
	}

	if shouldExit {
		return nil
	}

	// Create a context that is canceled on receiving an OS interrupt signal.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop() // Ensures the signal listener is cleaned up.

	// Step 1: Create the application with the signal-aware context.
	// If this call panics, execution jumps to the deferred function above.
	app := app.NewApp(ctx, outW, appConfig, nil)

	// Step 2: Run the application.
	// The error returned here will be the function's return value.
	err = app.Run()

	return err
}
