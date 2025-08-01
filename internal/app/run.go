package app

import (
	"context"
	"fmt"
	"reflect"

	"github.com/vk/burstgridgo/internal/ctxlog"
	"github.com/vk/burstgridgo/internal/dag"
	"github.com/vk/burstgridgo/internal/executor"
)

// Run executes the main application logic based on the provided configuration.
func (a *App) Run(ctx context.Context, appConfig *AppConfig) error {
	ctx = ctxlog.WithLogger(ctx, a.logger)
	a.logger.Debug("App.Run method started.")

	if appConfig.HealthcheckPort > 0 {
		go a.startHealthcheckServer(appConfig.HealthcheckPort)
	}

	a.logger.Debug("Building dependency graph from config model...")
	// Pass the pre-loaded, format-agnostic config model to the DAG builder.
	graph, err := dag.Build(ctx, a.config, a.registry)
	if err != nil {
		return fmt.Errorf("failed to build dependency graph: %w", err)
	}
	a.logger.Debug("Dependency graph built.", "node_count", len(graph.Nodes))

	a.logger.Info("Step handlers registered:", "count", len(a.registry.HandlerRegistry), "keys", reflect.ValueOf(a.registry.HandlerRegistry).MapKeys())
	a.logger.Info("Asset handlers registered:", "count", len(a.registry.AssetHandlerRegistry), "keys", reflect.ValueOf(a.registry.AssetHandlerRegistry).MapKeys())

	if len(graph.Nodes) > 0 {
		a.logger.Debug("Executor starting run.")
		a.logger.Info("🚀 Starting concurrent execution...")
		exec := executor.New(graph, appConfig.WorkerCount, a.registry, a.converter)
		if err := exec.Run(ctx); err != nil {
			return fmt.Errorf("execution failed: %w", err)
		}
		a.logger.Info("🏁 Execution finished.")
	} else {
		a.logger.Warn("No nodes found in graph, execution not required.")
	}

	a.logger.Debug("App.Run method finished.")
	return nil
}
