package builder

import (
	"context"
	"fmt"

	"github.com/specialistvlad/burstgridgo/internal/config"
	"github.com/specialistvlad/burstgridgo/internal/ctxlog"
	"github.com/specialistvlad/burstgridgo/internal/dag"
	"github.com/specialistvlad/burstgridgo/internal/node"
	"github.com/specialistvlad/burstgridgo/internal/registry"
)

// BuildStatic constructs a complete, validated dependency graph from a config model.
func BuildStatic(ctx context.Context, model *config.Model, r *registry.Registry) (*Storage, error) {
	logger := ctxlog.FromContext(ctx)
	logger.Debug("Build: Starting graph construction.")
	graph := &Storage{
		Nodes: make(map[string]*node.Node),
		dag:   dag.New(),
	}

	// First pass: create all Nodes for steps and resources.
	graph.createNodes(ctx, model.Grid)
	logger.Debug("Build: Node creation complete.", "Node_count", len(graph.Nodes))

	// Second pass: link dependencies.
	if err := graph.linkNodes(ctx, model, r); err != nil {
		return nil, err
	}
	logger.Debug("Build: Node linking complete.")

	// Third pass: initialize counters.
	logger.Debug("Build: Initializing Node counters from graph topology.")
	for _, Node := range graph.Nodes {
		if err := graph.SetInitialCounters(ctx, Node); err != nil {
			return nil, fmt.Errorf("failed to initialize counters for Node %s: %w", Node.ID(), err)
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
