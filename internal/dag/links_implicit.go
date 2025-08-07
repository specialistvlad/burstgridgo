package dag

import (
	"context"
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/vk/burstgridgo/internal/config"
	"github.com/vk/burstgridgo/internal/ctxlog"
	"github.com/vk/burstgridgo/internal/registry"
	"github.com/zclconf/go-cty/cty"
)

// parsedStepRef holds information extracted from an HCL traversal.
type parsedStepRef struct {
	FullName string // e.g., "runner_type.instance_name"
	Index    int    // The index accessed, or -1 if none.
}

// parseStepTraversal analyzes an HCL traversal to extract a step reference.
// A valid step reference is of the form `step.<runner_type>.<instance_name>`,
// optionally followed by an index.
func parseStepTraversal(traversal hcl.Traversal) (*parsedStepRef, bool) {
	if len(traversal) < 3 || traversal.RootName() != "step" {
		return nil, false
	}

	// Expect step.<runner_type>.<instance_name>
	runnerAttr, runnerOk := traversal[1].(hcl.TraverseAttr)
	nameAttr, nameOk := traversal[2].(hcl.TraverseAttr)
	if !runnerOk || !nameOk {
		return nil, false
	}

	fullName := fmt.Sprintf("%s.%s", runnerAttr.Name, nameAttr.Name)
	index := -1

	// Check if an index immediately follows the name.
	if len(traversal) > 3 {
		if indexer, ok := traversal[3].(hcl.TraverseIndex); ok {
			if indexer.Key.Type() == cty.Number {
				num := indexer.Key.AsBigFloat()
				if num.IsInt() {
					val, _ := num.Int64()
					index = int(val)
				}
			}
		}
	}

	return &parsedStepRef{
		FullName: fullName,
		Index:    index,
	}, true
}

// linkImplicitDeps parses an expression for variable traversals to create dependency links.
func linkImplicitDeps(ctx context.Context, node *Node, expr hcl.Expression, model *config.Model, graph *Graph, r *registry.Registry) error {
	baseLogger := ctxlog.FromContext(ctx)

	stepConfigMap := make(map[string]*config.Step)
	for _, step := range model.Grid.Steps {
		key := fmt.Sprintf("%s.%s", step.RunnerType, step.Name)
		stepConfigMap[key] = step
	}

	for _, traversal := range expr.Variables() {
		logger := baseLogger.With("node_id", node.ID, "traversal", formatTraversal(traversal))

		if ref, ok := parseStepTraversal(traversal); ok {
			logger.Debug("Parsed implicit dependency as step reference.", "ref_name", ref.FullName, "ref_index", ref.Index)

			depStepConfig, configFound := stepConfigMap[ref.FullName]
			if !configFound {
				logger.Debug("Traversal refers to an unknown step config, ignoring as dependency.")
				continue
			}

			finalIndex := ref.Index
			if finalIndex == -1 { // Shorthand reference
				logger.Debug("Handling shorthand implicit reference.", "instancing_mode", depStepConfig.Instancing)
				if depStepConfig.Instancing == config.ModeInstanced {
					return fmt.Errorf("ambiguous implicit dependency in '%s': expression refers to instanced step '%s' without an index", node.ID, ref.FullName)
				}
				finalIndex = 0 // It's a singular step, so default to [0]
			}

			depNodeID := fmt.Sprintf("step.%s[%d]", ref.FullName, finalIndex)
			depNode, nodeFound := graph.Nodes[depNodeID]
			if !nodeFound {
				return fmt.Errorf("implicit dependency error in '%s': referenced step instance '%s' does not exist", node.ID, depNodeID)
			}

			if err := validateOutputReference(traversal, depNode, r); err != nil {
				return err
			}

			if _, exists := node.Deps[depNodeID]; !exists {
				logger.Debug("Linking implicit dependency.", "from", node.ID, "to", depNodeID)
				node.Deps[depNodeID] = depNode
				depNode.Dependents[node.ID] = node
			}
			continue
		}

		// Fallback for resource dependencies
		if len(traversal) >= 3 && traversal.RootName() == "resource" {
			typeAttr, typeOk := traversal[1].(hcl.TraverseAttr)
			nameAttr, nameOk := traversal[2].(hcl.TraverseAttr)
			if typeOk && nameOk {
				depID := fmt.Sprintf("resource.%s.%s", typeAttr.Name, nameAttr.Name)
				if depNode, ok := graph.Nodes[depID]; ok {
					if _, exists := node.Deps[depID]; !exists {
						logger.Debug("Linking implicit dependency.", "from", node.ID, "to", depID)
						node.Deps[depID] = depNode
						depNode.Dependents[node.ID] = node
					}
				}
			}
		}
	}
	return nil
}
