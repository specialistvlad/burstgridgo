package main

import (
	"log/slog"
	"os"

	"github.com/hashicorp/hcl/v2"
	"github.com/vk/burstgridgo/internal/config"
	"github.com/vk/burstgridgo/internal/dag"
	"github.com/vk/burstgridgo/internal/engine"
	"github.com/vk/burstgridgo/internal/healthcheck"
	_ "github.com/vk/burstgridgo/modules/env_vars"
	_ "github.com/vk/burstgridgo/modules/help"
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

	// Start the health check server if the port is configured.
	if cliOpts.HealthcheckPort > 0 {
		go healthcheck.StartServer(cliOpts.HealthcheckPort)
	}

	var allModules []*engine.Module

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

			// Parse all files to get a flat list of modules.
			for _, file := range hclFiles {
				config, err := engine.DecodeHCLFile(file)
				if err != nil {
					slog.Warn("Failed to decode HCL file", "path", file, "error", err)
					continue
				}
				allModules = append(allModules, config.Modules...)
			}
		}
	}

	// 4. If no modules were loaded, inject the help module.
	if len(allModules) == 0 {
		if cliOpts.GridPath != "" {
			slog.Info("No modules found in path, displaying help.", "path", cliOpts.GridPath)
		}
		// Create a single module to run the help runner.
		allModules = []*engine.Module{
			{
				Name:   "show_help",
				Runner: "help",
				Body:   hcl.EmptyBody(),
			},
		}
	}

	// 5. Build the dependency graph.
	graph, err := dag.NewGraph(allModules)
	if err != nil {
		slog.Error("Failed to build dependency graph", "error", err)
		os.Exit(1)
	}

	// 6. Create an executor and run the graph.
	if len(graph.Nodes) > 0 {
		slog.Info("🚀 Starting concurrent execution...")
		executor := dag.NewExecutor(graph)
		if err := executor.Run(); err != nil {
			slog.Error("Execution failed", "error", err)
			os.Exit(1)
		}
		slog.Info("🏁 Execution finished.")
	}
}
