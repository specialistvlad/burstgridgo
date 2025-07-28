package app

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"reflect"

	"github.com/vk/burstgridgo/internal/ctxlog"
	"github.com/vk/burstgridgo/internal/dag"
	"github.com/vk/burstgridgo/internal/engine"
	"github.com/vk/burstgridgo/internal/registry"
)

// AppConfig holds all the necessary configuration for an App instance to run.
type AppConfig struct {
	GridPath        string
	ModulesPath     string
	HealthcheckPort int
	LogFormat       string
	LogLevel        string
	WorkerCount     int
}

// App encapsulates the application's dependencies, configuration, and lifecycle.
type App struct {
	outW     io.Writer
	logger   *slog.Logger
	registry *registry.Registry
}

// New is the constructor for the main application. It returns a fully
// initialized App instance, including its own isolated logger and registry.
// It accepts an optional list of modules to register, which is used for testing.
// If no modules are provided, it defaults to the core application modules.
func New(outW io.Writer, appConfig *AppConfig, modules ...registry.Module) *App {
	logger := newLogger(appConfig.LogLevel, appConfig.LogFormat, outW)
	logger.Debug("Logger configured successfully.")

	reg := registry.New()
	// If no modules are passed, use the default core modules.
	if len(modules) == 0 {
		modules = coreModules
	}

	for _, mod := range modules {
		mod.Register(reg)
	}
	logger.Debug("All modules registered.", "count", len(modules))

	return &App{
		outW:     outW,
		logger:   logger,
		registry: reg,
	}
}

// Run executes the main application logic based on the provided configuration.
func (a *App) Run(ctx context.Context, appConfig *AppConfig) error {
	// Create a new context that carries the application-specific logger.
	ctx = ctxlog.WithLogger(ctx, a.logger)
	a.logger.Debug("App.Run method started.")

	if appConfig.ModulesPath != "" {
		a.logger.Debug("Starting module discovery...", "path", appConfig.ModulesPath)
		a.logger.Info("Discovering available modules (runners & assets)...")
		if err := engine.DiscoverModules(ctx, appConfig.ModulesPath, a.registry); err != nil {
			return fmt.Errorf("failed to discover modules: %w", err)
		}
		a.logger.Debug("Module discovery complete.")
	} else {
		a.logger.Warn("No modules path provided, skipping module discovery.")
	}

	if appConfig.HealthcheckPort > 0 {
		a.logger.Debug("Health check server configured.", "port", appConfig.HealthcheckPort)
		go a.startHealthcheckServer(appConfig.HealthcheckPort)
	}

	a.logger.Debug("Loading grid configuration.", "path", appConfig.GridPath)
	gridConfig, err := engine.LoadGridConfig(ctx, appConfig.GridPath)
	if err != nil {
		return fmt.Errorf("failed to load grid configuration: %w", err)
	}
	a.logger.Debug("Grid configuration loaded successfully.")

	a.logger.Debug("Building dependency graph...")
	graph, err := dag.NewGraph(ctx, gridConfig)
	if err != nil {
		return fmt.Errorf("failed to build dependency graph: %w", err)
	}
	a.logger.Debug("Dependency graph built.", "node_count", len(graph.Nodes))

	a.logger.Info("Step handlers registered:", "count", len(a.registry.HandlerRegistry), "keys", reflect.ValueOf(a.registry.HandlerRegistry).MapKeys())
	a.logger.Info("Asset handlers registered:", "count", len(a.registry.AssetHandlerRegistry), "keys", reflect.ValueOf(a.registry.AssetHandlerRegistry).MapKeys())

	if len(graph.Nodes) > 0 {
		a.logger.Debug("Executor starting run.")
		a.logger.Info("🚀 Starting concurrent execution...")
		executor := dag.NewExecutor(graph, appConfig.WorkerCount, a.registry)
		if err := executor.Run(ctx); err != nil { // Pass the logger-aware context
			return fmt.Errorf("execution failed: %w", err)
		}
		a.logger.Info("🏁 Execution finished.")
	} else {
		a.logger.Warn("No nodes found in graph, execution not required.")
	}

	a.logger.Debug("App.Run method finished.")
	return nil
}
