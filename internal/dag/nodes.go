package dag

import (
	"context"
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/vk/burstgridgo/internal/config"
	"github.com/vk/burstgridgo/internal/ctxlog"
	"github.com/vk/burstgridgo/internal/registry"
)

// createNodes performs the first pass of graph creation.
func createNodes(ctx context.Context, grid *config.Grid, graph *Graph) {
	logger := ctxlog.FromContext(ctx)
	for _, s := range grid.Steps {
		expandedSteps := expandStep(s)
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
				Deps:       make(map[string]*Node),
				Dependents: make(map[string]*Node),
			}
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

// linkNodes performs the second pass, establishing dependency links.
func linkNodes(ctx context.Context, model *config.Model, graph *Graph, r *registry.Registry) error {
	logger := ctxlog.FromContext(ctx)
	logger.Debug("Starting node linking pass.")

	for _, node := range graph.Nodes {
		nodeLogger := logger.With("node_id", node.ID)
		nodeLogger.Debug("Processing dependencies for node.")
		var dependsOn []string
		var expressions []hcl.Expression

		if node.Type == StepNode {
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

		if err := linkExplicitDeps(ctx, node, dependsOn, model, graph); err != nil {
			return err
		}
		for _, expr := range expressions {
			// Pass model to implicit linker to make it instance-aware.
			if err := linkImplicitDeps(ctx, node, expr, model, graph, r); err != nil {
				return err
			}
		}
	}
	logger.Debug("Finished node linking pass.")
	return nil
}
