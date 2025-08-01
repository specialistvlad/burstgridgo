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
	if err := linkNodes(ctx, graph, r); err != nil {
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

// createNodes performs the first pass of graph creation.
func createNodes(ctx context.Context, grid *config.Grid, graph *Graph) {
	logger := ctxlog.FromContext(ctx)
	for _, s := range grid.Steps {
		id := fmt.Sprintf("step.%s.%s", s.RunnerType, s.Name)
		if _, exists := graph.Nodes[id]; exists {
			logger.Warn("Duplicate step definition found, it will be overwritten.", "id", id)
		}
		graph.Nodes[id] = &Node{
			ID:         id,
			Name:       s.Name,
			Type:       StepNode,
			StepConfig: s,
			Deps:       make(map[string]*Node),
			Dependents: make(map[string]*Node),
		}
	}
	for _, r := range grid.Resources {
		id := fmt.Sprintf("resource.%s.%s", r.AssetType, r.Name)
		if _, exists := graph.Nodes[id]; exists {
			logger.Warn("Duplicate resource definition found, it will be overwritten.", "id", id)
		}
		graph.Nodes[id] = &Node{
			ID:             id,
			Name:           r.Name,
			Type:           ResourceNode,
			ResourceConfig: r,
			Deps:           make(map[string]*Node),
			Dependents:     make(map[string]*Node),
		}
	}
}

// detectCycles checks for circular dependencies in the graph using DFS.
func (g *Graph) detectCycles() error {
	visiting := make(map[string]bool)
	visited := make(map[string]bool)

	var visit func(node *Node) error
	visit = func(node *Node) error {
		visiting[node.ID] = true
		for _, dep := range node.Deps {
			if visiting[dep.ID] {
				return fmt.Errorf("cycle detected involving '%s'", dep.ID)
			}
			if !visited[dep.ID] {
				if err := visit(dep); err != nil {
					return err
				}
			}
		}
		delete(visiting, node.ID)
		visited[node.ID] = true
		return nil
	}

	for _, node := range g.Nodes {
		if !visited[node.ID] {
			if err := visit(node); err != nil {
				return err
			}
		}
	}
	return nil
}
