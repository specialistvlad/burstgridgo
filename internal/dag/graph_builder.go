package dag

import (
	"context"
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/vk/burstgridgo/internal/ctxlog"
	"github.com/vk/burstgridgo/internal/schema"
)

// createNodes performs the first pass of graph creation.
func createNodes(ctx context.Context, config *schema.GridConfig, graph *Graph) {
	logger := ctxlog.FromContext(ctx)
	for _, s := range config.Steps {
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
	for _, r := range config.Resources {
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
func linkNodes(ctx context.Context, graph *Graph) error {
	for _, node := range graph.Nodes {
		var dependsOn []string
		var bodies []hcl.Body
		if node.Type == StepNode {
			dependsOn = node.StepConfig.DependsOn
			if node.StepConfig.Arguments != nil && node.StepConfig.Arguments.Body != nil {
				bodies = append(bodies, node.StepConfig.Arguments.Body)
			}
			if node.StepConfig.Uses != nil && node.StepConfig.Uses.Body != nil {
				bodies = append(bodies, node.StepConfig.Uses.Body)
			}
		} else { // ResourceNode
			dependsOn = node.ResourceConfig.DependsOn
			if node.ResourceConfig.Arguments != nil && node.ResourceConfig.Arguments.Body != nil {
				bodies = append(bodies, node.ResourceConfig.Arguments.Body)
			}
		}

		if err := linkExplicitDeps(ctx, node, dependsOn, graph); err != nil {
			return err
		}
		for _, body := range bodies {
			if err := linkImplicitDeps(ctx, node, body, graph); err != nil {
				return err
			}
		}
	}
	return nil
}

// initializeCounters performs the third pass, calculating initial counters.
func initializeCounters(graph *Graph) {
	for _, node := range graph.Nodes {
		node.depCount.Store(int32(len(node.Deps)))
		if node.Type == ResourceNode {
			var directStepDependents int32 = 0
			// THIS IS THE FIX: Iterate through the dependents and only count
			// the ones that are actual steps. This is the key to efficient cleanup.
			for _, dependent := range node.Dependents {
				if dependent.Type == StepNode {
					directStepDependents++
				}
			}
			node.descendantCount.Store(directStepDependents)
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

// linkExplicitDeps resolves dependencies from a `depends_on` block.
func linkExplicitDeps(ctx context.Context, node *Node, dependsOn []string, graph *Graph) error {
	logger := ctxlog.FromContext(ctx)
	for _, depAddr := range dependsOn {
		stepID := "step." + depAddr
		resourceID := "resource." + depAddr

		var depNode *Node
		var found bool
		if depNode, found = graph.Nodes[stepID]; !found {
			if depNode, found = graph.Nodes[resourceID]; !found {
				return fmt.Errorf("node '%s' depends on non-existent identifier '%s'", node.ID, depAddr)
			}
		}

		if _, exists := node.Deps[depNode.ID]; !exists {
			logger.Debug("Linking explicit dependency.", "from", node.ID, "to", depNode.ID)
			node.Deps[depNode.ID] = depNode
			depNode.Dependents[node.ID] = node
		}
	}
	return nil
}

// linkImplicitDeps parses a body for variable traversals to create dependency links.
func linkImplicitDeps(ctx context.Context, node *Node, body hcl.Body, graph *Graph) error {
	logger := ctxlog.FromContext(ctx)
	attrs, diags := body.JustAttributes()
	if diags.HasErrors() {
		return diags
	}

	for _, attr := range attrs {
		for _, traversal := range attr.Expr.Variables() {
			if len(traversal) < 3 {
				continue
			}
			rootName := traversal.RootName()
			if rootName != "step" && rootName != "resource" {
				continue
			}
			typeAttr, typeOk := traversal[1].(hcl.TraverseAttr)
			nameAttr, nameOk := traversal[2].(hcl.TraverseAttr)
			if !typeOk || !nameOk {
				continue
			}
			depID := fmt.Sprintf("%s.%s.%s", rootName, typeAttr.Name, nameAttr.Name)
			depNode, ok := graph.Nodes[depID]
			if !ok {
				continue
			}
			if _, exists := node.Deps[depID]; !exists {
				logger.Debug("Linking implicit dependency.", "from", node.ID, "to", depID)
				node.Deps[depID] = depNode
				depNode.Dependents[node.ID] = node
			}
		}
	}
	return nil
}
