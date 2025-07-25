package main

import (
	"log/slog"
	"os"
	"reflect"

	"github.com/vk/burstgridgo/internal/config"
	"github.com/vk/burstgridgo/internal/dag"
	"github.com/vk/burstgridgo/internal/engine"
	"github.com/vk/burstgridgo/internal/healthcheck"
	_ "github.com/vk/burstgridgo/modules/counter_op"
	_ "github.com/vk/burstgridgo/modules/env_vars"
	_ "github.com/vk/burstgridgo/modules/help"
	_ "github.com/vk/burstgridgo/modules/http_client"
	_ "github.com/vk/burstgridgo/modules/http_request"
	_ "github.com/vk/burstgridgo/modules/local_counter"
	_ "github.com/vk/burstgridgo/modules/print"
	_ "github.com/vk/burstgridgo/modules/s3"
	_ "github.com/vk/burstgridgo/modules/socketio"
	_ "github.com/vk/burstgridgo/modules/socketio_client"
	_ "github.com/vk/burstgridgo/modules/socketio_request"
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

	// 3. Load, parse, and merge the grid configuration from the specified path.
	// This single call replaces the previous block of file handling logic.
	gridConfig, err := engine.LoadGridConfig(cliOpts.GridPath)
	if err != nil {
		slog.Error("Failed to load grid configuration", "error", err)
		os.Exit(1)
	}

	// 4. Build the dependency graph.
	graph, err := dag.NewGraph(gridConfig)
	if err != nil {
		slog.Error("Failed to build dependency graph", "error", err)
		os.Exit(1)
	}

	slog.Info("Step handlers registered:", "count", len(engine.HandlerRegistry), "keys", reflect.ValueOf(engine.HandlerRegistry).MapKeys())
	slog.Info("Asset handlers registered:", "count", len(engine.AssetHandlerRegistry), "keys", reflect.ValueOf(engine.AssetHandlerRegistry).MapKeys())

	// 5. Create an executor and run the graph.
	if len(graph.Nodes) > 0 {
		slog.Info("🚀 Starting concurrent execution...")
		executor := dag.NewExecutor(graph, nil, nil)
		if err := executor.Run(); err != nil {
			slog.Error("Execution failed", "error", err)
			os.Exit(1)
		}
		slog.Info("🏁 Execution finished.")
	}
}
