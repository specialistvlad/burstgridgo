package builder

import (
	"context"
	"fmt"

	"github.com/vk/burstgridgo/internal/config"
	"github.com/vk/burstgridgo/internal/ctxlog"
	"github.com/vk/burstgridgo/internal/dag"
	"github.com/vk/burstgridgo/internal/registry"
)

// BuildStatic constructs a complete, validated dependency graph from a config model.
func BuildStatic(ctx context.Context, model *config.Model, r *registry.Registry) (*Graph, error) {
	logger := ctxlog.FromContext(ctx)
	logger.Debug("Build: Starting graph construction.")
	graph := &Graph{
		Nodes: make(map[string]*Node),
		dag:   dag.New(),
	}

	// First pass: create all nodes for steps and resources.
	createNodes(ctx, model.Grid, graph)
	logger.Debug("Build: Node creation complete.", "node_count", len(graph.Nodes))

	// Second pass: link dependencies.
	if err := linkNodes(ctx, model, graph, r); err != nil {
		return nil, err
	}
	logger.Debug("Build: Node linking complete.")

	// Third pass: initialize counters.
	logger.Debug("Build: Initializing node counters from graph topology.")
	for _, node := range graph.Nodes {
		if err := node.SetInitialCounters(ctx, graph); err != nil {
			return nil, fmt.Errorf("failed to initialize counters for node %s: %w", node.ID, err)
		}
	}
	logger.Debug("Build: Counter initialization complete.")

	// Final validation: Cycle detection.
	if err := graph.dag.DetectCycles(); err != nil {
		return nil, fmt.Errorf("error validating dependency graph: %w", err)
	}
	logger.Debug("Build: Cycle detection passed.")

	logger.Info("Build: Graph construction successful.")
	return graph, nil
}
