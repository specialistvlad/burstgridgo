package config

import (
	"flag"
	"fmt"
	"os"
)

// CLIOptions holds the configuration parsed from the command line.
type CLIOptions struct {
	GridPath string
}

// Parse processes the command-line arguments and returns structured options.
func Parse() (*CLIOptions, error) {
	// Set a custom usage message
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] [GRID_PATH]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Arguments:\n")
		fmt.Fprintf(os.Stderr, "  GRID_PATH\n")
		fmt.Fprintf(os.Stderr, "    	Path to a single .hcl file or a directory of .hcl files.\n")
		fmt.Fprintf(os.Stderr, "    	If not provided, defaults to the current directory '.'.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}

	// Define flags. We use separate variables to check which was set.
	gridFlag := flag.String("grid", "", "Path to the grid file or directory.")
	gFlag := flag.String("g", "", "Path to the grid file or directory (shorthand).")

	flag.Parse()

	// Determine the path based on precedence: --grid, -g, positional arg, default.
	path := "." // Default path
	if *gridFlag != "" {
		path = *gridFlag
	} else if *gFlag != "" {
		path = *gFlag
	} else if flag.NArg() > 0 {
		path = flag.Arg(0)
	}

	return &CLIOptions{GridPath: path}, nil
}
