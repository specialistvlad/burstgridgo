package main

import (
	"log/slog"
	"os"
	"reflect"

	"github.com/hashicorp/hcl/v2"
	"github.com/vk/burstgridgo/internal/config"
	"github.com/vk/burstgridgo/internal/dag"
	"github.com/vk/burstgridgo/internal/engine"
	"github.com/vk/burstgridgo/internal/healthcheck"
	_ "github.com/vk/burstgridgo/modules/env_vars"
	_ "github.com/vk/burstgridgo/modules/help"
	_ "github.com/vk/burstgridgo/modules/http_client"
	_ "github.com/vk/burstgridgo/modules/http_request"
	_ "github.com/vk/burstgridgo/modules/print"
	_ "github.com/vk/burstgridgo/modules/s3"
	_ "github.com/vk/burstgridgo/modules/socketio"
)

func main() {
	// 1. Parse all CLI arguments and flags.
	cliOpts, err := config.Parse()
	if err != nil {
		slog.Error("Failed to parse arguments", "error", err)
		os.Exit(1)
	}

	// 2. Initialize the structured logger.
	var level slog.Level
	switch cliOpts.LogLevel {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}

	handlerOpts := &slog.HandlerOptions{Level: level}
	var handler slog.Handler
	if cliOpts.LogFormat == "json" {
		handler = slog.NewJSONHandler(os.Stdout, handlerOpts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, handlerOpts)
	}
	slog.SetDefault(slog.New(handler))

	// Discover all available module definitions from the modules directory.
	slog.Info("Discovering available modules (runners & assets)...")
	if err := engine.DiscoverModules("modules"); err != nil {
		slog.Error("Failed to discover modules", "error", err)
		os.Exit(1)
	}

	// Start the health check server if the port is configured.
	if cliOpts.HealthcheckPort > 0 {
		go healthcheck.StartServer(cliOpts.HealthcheckPort)
	}

	// This will hold the combined configuration from all parsed files.
	gridConfig := &engine.GridConfig{}

	// 3. If a grid path is provided, find and parse the HCL files.
	if cliOpts.GridPath != "" {
		hclFiles, err := engine.ResolveGridPath(cliOpts.GridPath)
		if err != nil {
			slog.Error("Failed to resolve grid path", "path", cliOpts.GridPath, "error", err)
			os.Exit(1)
		}

		if len(hclFiles) > 0 {
			slog.Info("Found HCL files to process", "count", len(hclFiles), "path", cliOpts.GridPath)
			for _, file := range hclFiles {
				slog.Debug("Resolved HCL file", "path", file)
			}

			// Parse all files and merge them into a single config.
			for _, file := range hclFiles {
				cfg, err := engine.DecodeGridFile(file)
				if err != nil {
					slog.Warn("Failed to decode HCL file", "path", file, "error", err)
					continue
				}
				// Append resources and steps from the parsed file.
				gridConfig.Resources = append(gridConfig.Resources, cfg.Resources...)
				gridConfig.Steps = append(gridConfig.Steps, cfg.Steps...)
			}
		}
	}

	// 4. If no steps were loaded, inject the help step.
	if len(gridConfig.Steps) == 0 {
		if cliOpts.GridPath != "" {
			slog.Info("No steps found in path, displaying help.", "path", cliOpts.GridPath)
		}
		// Create a single step to run the help runner.
		gridConfig.Steps = []*engine.Step{
			{
				Name:       "show_help",
				RunnerType: "help",
				Arguments: &engine.StepArgs{ // Correctly initialize StepArgs struct
					Body: hcl.EmptyBody(), // with an empty HCL body
				},
			},
		}
	}

	// 5. Build the dependency graph.
	graph, err := dag.NewGraph(gridConfig)
	if err != nil {
		slog.Error("Failed to build dependency graph", "error", err)
		os.Exit(1)
	}

	slog.Info("Step handlers registered:", "count", len(engine.HandlerRegistry), "keys", reflect.ValueOf(engine.HandlerRegistry).MapKeys())
	slog.Info("Asset handlers registered:", "count", len(engine.AssetHandlerRegistry), "keys", reflect.ValueOf(engine.AssetHandlerRegistry).MapKeys())

	// 6. Create an executor and run the graph.
	if len(graph.Nodes) > 0 {
		slog.Info("🚀 Starting concurrent execution...")
		executor := dag.NewExecutor(graph, nil)
		if err := executor.Run(); err != nil {
			slog.Error("Execution failed", "error", err)
			os.Exit(1)
		}
		slog.Info("🏁 Execution finished.")
	}
}
