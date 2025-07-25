package engine

import (
	"fmt"
	"log/slog"

	"github.com/hashicorp/hcl/v2"
)

// LoadGridConfig encapsulates the logic for finding, parsing, merging,
// and preparing a GridConfig from a given path. It also injects the default
// 'help' step if no other steps are found.
func LoadGridConfig(gridPath string) (*GridConfig, error) {
	// This will hold the combined configuration from all parsed files.
	gridConfig := &GridConfig{}

	// If a grid path is provided, find and parse the HCL files.
	if gridPath != "" {
		hclFiles, err := ResolveGridPath(gridPath)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve grid path '%s': %w", gridPath, err)
		}

		if len(hclFiles) > 0 {
			slog.Info("Found HCL files to process", "count", len(hclFiles), "path", gridPath)
			for _, file := range hclFiles {
				slog.Debug("Resolved HCL file", "path", file)
			}

			// Parse all files and merge them into a single config.
			for _, file := range hclFiles {
				cfg, err := DecodeGridFile(file)
				if err != nil {
					// Log a warning for a single bad file but continue processing others.
					slog.Warn("Failed to decode HCL file, skipping", "path", file, "error", err)
					continue
				}
				// Append resources and steps from the parsed file.
				gridConfig.Resources = append(gridConfig.Resources, cfg.Resources...)
				gridConfig.Steps = append(gridConfig.Steps, cfg.Steps...)
			}
		}
	}

	// If no steps were loaded from files, inject the default help step.
	if len(gridConfig.Steps) == 0 {
		if gridPath != "" {
			slog.Info("No steps found in path, displaying help.", "path", gridPath)
		}
		// Create a single step to run the help runner.
		gridConfig.Steps = []*Step{
			{
				Name:       "show_help",
				RunnerType: "help",
				Arguments: &StepArgs{
					Body: hcl.EmptyBody(),
				},
			},
		}
	}

	return gridConfig, nil
}
