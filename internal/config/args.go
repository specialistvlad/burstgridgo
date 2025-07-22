package config

import (
	"flag"
)

// CLIOptions holds the configuration parsed from the command line.
type CLIOptions struct {
	GridPath        string
	HealthcheckPort int
}

// Parse processes the command-line arguments and returns structured options.
func Parse() (*CLIOptions, error) {
	// Define flags.
	gridFlag := flag.String("grid", "", "Path to the grid file or directory.")
	gFlag := flag.String("g", "", "Path to the grid file or directory (shorthand).")
	healthPortFlag := flag.Int("healthcheck-port", 8080, "Port for the HTTP health check server. Set to 0 to disable.")

	flag.Parse()

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
	}, nil
}
