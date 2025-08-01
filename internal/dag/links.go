package dag

import (
	"context"
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/vk/burstgridgo/internal/ctxlog" // Corrected import path
	"github.com/vk/burstgridgo/internal/registry"
)

// linkNodes performs the second pass, establishing dependency links.
func linkNodes(ctx context.Context, graph *Graph, r *registry.Registry) error {
	for _, node := range graph.Nodes {
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

		if err := linkExplicitDeps(ctx, node, dependsOn, graph); err != nil {
			return err
		}
		for _, expr := range expressions {
			if err := linkImplicitDeps(ctx, node, expr, graph, r); err != nil {
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

// linkImplicitDeps parses an expression for variable traversals to create dependency links.
func linkImplicitDeps(ctx context.Context, node *Node, expr hcl.Expression, graph *Graph, r *registry.Registry) error {
	logger := ctxlog.FromContext(ctx)
	for _, traversal := range expr.Variables() {
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
			// This could be a reference to something else, like a variable.
			continue
		}

		// If referencing an output, validate it exists in the manifest.
		if len(traversal) > 3 {
			if outputAttr, isOutput := traversal[3].(hcl.TraverseAttr); isOutput && outputAttr.Name == "output" {
				if err := validateOutputReference(traversal, depNode, r); err != nil {
					return err
				}
			}
		}

		if _, exists := node.Deps[depID]; !exists {
			logger.Debug("Linking implicit dependency.", "from", node.ID, "to", depID)
			node.Deps[depID] = depNode
			depNode.Dependents[node.ID] = node
		}
	}
	return nil
}

// validateOutputReference checks if a reference to a step's output is valid.
func validateOutputReference(traversal hcl.Traversal, depNode *Node, r *registry.Registry) error {
	if depNode.Type != StepNode || len(traversal) < 5 {
		return nil // Not a step output reference we need to validate.
	}

	outputNameAttr, ok := traversal[4].(hcl.TraverseAttr)
	if !ok {
		return nil // Malformed traversal.
	}
	outputName := outputNameAttr.Name

	runnerDef, ok := r.DefinitionRegistry[depNode.StepConfig.RunnerType]
	if !ok {
		return fmt.Errorf("internal error: could not find definition for runner type %s", depNode.StepConfig.RunnerType)
	}

	if _, ok := runnerDef.Outputs[outputName]; ok {
		return nil // Found a valid declaration.
	}

	return fmt.Errorf("reference to undeclared output %q on step %q", outputName, depNode.ID)
}
