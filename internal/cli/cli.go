package cli

import (
	"flag"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/specialistvlad/burstgridgo/internal/app"
)

// ExitError is a custom error type that includes a specific exit code.
type ExitError struct {
	Code    int
	Message string
}

// Error implements the error interface for ExitError.
func (e *ExitError) Error() string {
	return e.Message
}

// Parse processes command-line arguments. It returns a populated AppConfig,
// a boolean indicating if the program should exit cleanly, or an ExitError.
func Parse(args []string, output io.Writer) (*app.Config, bool, error) {
	slog.Debug("CLI parser started.")
	flagSet := flag.NewFlagSet("burstgridgo", flag.ContinueOnError)
	flagSet.SetOutput(output)

	// Custom usage/help text function
	flagSet.Usage = func() {
		fmt.Fprint(output, `
BurstGridGo - A declarative, concurrency-first load testing tool.

Usage:
  burstgridgo [options] [GRID_PATH]

Arguments:
  GRID_PATH
    Path to a single .hcl file or a directory containing .hcl files.

Options:
`)
		flagSet.PrintDefaults()
	}

	gridFlag := flagSet.String("grid", "", "Path to the grid file or directory.")
	gFlag := flagSet.String("g", "", "Path to the grid file or directory (shorthand).")
	healthPortFlag := flagSet.Int("healthcheck-port", 0, "Port for the HTTP health check server. 0 is disabled.")
	logFormatFlag := flagSet.String("log-format", "json", "Log output format. Options: 'text' or 'json'.")
	logLevelFlag := flagSet.String("log-level", "info", "Set the logging level. Options: 'debug', 'info', 'warn', 'error'.")
	workersFlag := flagSet.Int("workers", 10, "Number of concurrent workers for the executor.")
	modulesPathFlag := flagSet.String("modules-path", "modules", "Path to the directory containing module definitions.")

	if err := flagSet.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return nil, true, nil
		}
		return nil, false, &ExitError{Code: 2, Message: err.Error()}
	}
	slog.Debug("Arguments parsed successfully.")

	path := ""
	if *gridFlag != "" {
		path = *gridFlag
	} else if *gFlag != "" {
		path = *gFlag
	} else if flagSet.NArg() > 0 {
		path = flagSet.Arg(0)
	}
	slog.Debug("Grid path determined.", "path", path)

	if path == "" {
		slog.Debug("No grid path provided, printing usage and exiting.")
		flagSet.Usage()
		return nil, true, nil
	}

	logFormat := strings.ToLower(*logFormatFlag)
	if logFormat != "text" && logFormat != "json" {
		return nil, false, &ExitError{Code: 2, Message: "invalid log-format: must be 'text' or 'json'"}
	}

	logLevel := strings.ToLower(*logLevelFlag)
	switch logLevel {
	case "debug", "info", "warn", "error":
		// valid
	default:
		return nil, false, &ExitError{Code: 2, Message: "invalid log-level: must be 'debug', 'info', 'warn', or 'error'"}
	}
	slog.Debug("CLI parameter validation complete.")

	config, err := app.NewConfig(app.Config{
		GridPath:        path,
		ModulesPath:     *modulesPathFlag,
		HealthcheckPort: *healthPortFlag,
		LogFormat:       logFormat,
		LogLevel:        logLevel,
		WorkerCount:     *workersFlag,
	})

	if err != nil {
		return nil, false, &ExitError{Code: 2, Message: err.Error()}
	}

	slog.Debug("CLI parser finished successfully.", "config", config)
	return config, false, nil
}
