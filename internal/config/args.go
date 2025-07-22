package config

import (
	"flag"
	"fmt"
	"strings"
)

// CLIOptions holds the configuration parsed from the command line.
type CLIOptions struct {
	GridPath        string
	HealthcheckPort int
	LogFormat       string
}

// Parse processes the command-line arguments and returns structured options.
func Parse() (*CLIOptions, error) {
	// Define flags.
	gridFlag := flag.String("grid", "", "Path to the grid file or directory.")
	gFlag := flag.String("g", "", "Path to the grid file or directory (shorthand).")
	healthPortFlag := flag.Int("healthcheck-port", 8080, "Port for the HTTP health check server. Set to 0 to disable.")
	logFormatFlag := flag.String("log-format", "text", "Log output format. Options: 'text' or 'json'.")

	flag.Parse()

	// Validate log-format
	logFormat := strings.ToLower(*logFormatFlag)
	if logFormat != "text" && logFormat != "json" {
		return nil, fmt.Errorf("invalid log-format: must be 'text' or 'json'")
	}

	// Determine the path based on precedence: --grid, -g, positional arg.
	path := "" // Default path is empty.
	if *gridFlag != "" {
		path = *gridFlag
	} else if *gFlag != "" {
		path = *gFlag
	} else if flag.NArg() > 0 {
		path = flag.Arg(0)
	}

	return &CLIOptions{
		GridPath:        path,
		HealthcheckPort: *healthPortFlag,
		LogFormat:       logFormat,
	}, nil
}
