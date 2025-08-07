package dag

import (
	"context"
	"fmt"

	"github.com/vk/burstgridgo/internal/config"
	"github.com/vk/burstgridgo/internal/ctxlog"
	"github.com/vk/burstgridgo/internal/registry"
)

// Build constructs a complete, validated dependency graph from a config model.
func Build(ctx context.Context, model *config.Model, r *registry.Registry) (*Graph, error) {
	logger := ctxlog.FromContext(ctx)
	logger.Debug("Build: Starting graph construction.")
	graph := &Graph{Nodes: make(map[string]*Node)}

	// First pass: create all nodes for steps and resources.
	createNodes(ctx, model.Grid, graph)
	logger.Debug("Build: Node creation complete.", "node_count", len(graph.Nodes))

	// Second pass: link dependencies.
	// We now pass the model in to give the linkers configuration context.
	if err := linkNodes(ctx, model, graph, r); err != nil {
		return nil, err
	}
	logger.Debug("Build: Node linking complete.")

	// Third pass: initialize counters.
	for _, node := range graph.Nodes {
		node.SetInitialCounters()
	}
	logger.Debug("Build: Counter initialization complete.")

	if err := graph.detectCycles(); err != nil {
		return nil, fmt.Errorf("error validating dependency graph: %w", err)
	}
	logger.Debug("Build: Cycle detection passed.")

	logger.Debug("Build: Graph construction successful.")
	return graph, nil
}
