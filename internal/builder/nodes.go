package builder

import (
	"context"
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/vk/burstgridgo/internal/config"
	"github.com/vk/burstgridgo/internal/ctxlog"
	"github.com/vk/burstgridgo/internal/registry"
)

// createNodes performs the first pass of graph creation, populating the graph
// with all nodes defined in the configuration.
func createNodes(ctx context.Context, grid *config.Grid, graph *Graph) {
	logger := ctxlog.FromContext(ctx)
	logger.Debug("Starting node creation pass.")

	for _, s := range grid.Steps {
		expandedSteps, isPlaceholder := expandStep(s)

		if isPlaceholder {
			id := fmt.Sprintf("step.%s.%s", s.RunnerType, s.Name)
			logger.Debug("Creating placeholder step node.", "id", id)
			if _, exists := graph.Nodes[id]; exists {
				logger.Warn("Duplicate step definition found, it will be overwritten.", "id", id)
			}
			graph.Nodes[id] = &Node{
				ID:            id,
				Name:          s.Name,
				Type:          StepNode,
				IsPlaceholder: true,
				StepConfig:    s,
			}
			graph.dag.AddNode(id)
		} else {
			logger.Debug("Creating static step nodes.", "name", s.Name, "instance_count", len(expandedSteps))
			for i, expandedS := range expandedSteps {
				id := fmt.Sprintf("step.%s.%s[%d]", expandedS.RunnerType, expandedS.Name, i)
				if _, exists := graph.Nodes[id]; exists {
					logger.Warn("Duplicate step definition found, it will be overwritten.", "id", id)
				}
				graph.Nodes[id] = &Node{
					ID:         id,
					Name:       expandedS.Name,
					Type:       StepNode,
					StepConfig: expandedS,
				}
				graph.dag.AddNode(id)
			}
		}
	}
	for _, r := range grid.Resources {
		id := fmt.Sprintf("resource.%s.%s", r.AssetType, r.Name)
		logger.Debug("Creating resource node.", "id", id)
		if _, exists := graph.Nodes[id]; exists {
			logger.Warn("Duplicate resource definition found, it will be overwritten.", "id", id)
		}
		graph.Nodes[id] = &Node{
			ID:             id,
			Name:           r.Name,
			Type:           ResourceNode,
			ResourceConfig: r,
		}
		graph.dag.AddNode(id)
	}
	logger.Debug("Finished node creation pass.")
}

// linkNodes performs the second pass, establishing dependency edges between nodes.
func linkNodes(ctx context.Context, model *config.Model, graph *Graph, r *registry.Registry) error {
	logger := ctxlog.FromContext(ctx)
	logger.Debug("Starting node linking pass.")

	for _, node := range graph.Nodes {
		nodeLogger := logger.With("node_id", node.ID)
		nodeLogger.Debug("Processing dependencies for node.")
		var dependsOn []string
		var expressions []hcl.Expression

		if node.Type == StepNode {
			if node.IsPlaceholder && node.StepConfig.Count != nil {
				expressions = append(expressions, node.StepConfig.Count)
			}
			dependsOn = node.StepConfig.DependsOn
			for _, expr := range node.StepConfig.Arguments {
				expressions = append(expressions, expr)
			}
			for _, expr := range node.StepConfig.Uses {
				expressions = append(expressions, expr)
			}
		} else { // ResourceNode
			dependsOn = node.ResourceConfig.DependsOn
			for _, expr := range node.ResourceConfig.Arguments {
				expressions = append(expressions, expr)
			}
		}

		if len(dependsOn) > 0 {
			nodeLogger.Debug("Linking explicit dependencies.", "count", len(dependsOn))
			if err := linkExplicitDeps(ctx, node, dependsOn, model, graph); err != nil {
				return err
			}
		}

		if len(expressions) > 0 {
			nodeLogger.Debug("Linking implicit dependencies from expressions.", "count", len(expressions))
			for _, expr := range expressions {
				if err := linkImplicitDeps(ctx, node, expr, model, graph, r); err != nil {
					return err
				}
			}
		}
	}
	logger.Debug("Finished node linking pass.")
	return nil
}
