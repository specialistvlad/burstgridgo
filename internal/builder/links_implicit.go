package builder

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
func parseStepTraversal(traversal hcl.Traversal) (*parsedStepRef, bool) {
	if len(traversal) < 3 || traversal.RootName() != "step" {
		return nil, false
	}

	runnerAttr, runnerOk := traversal[1].(hcl.TraverseAttr)
	nameAttr, nameOk := traversal[2].(hcl.TraverseAttr)
	if !runnerOk || !nameOk {
		return nil, false
	}

	fullName := fmt.Sprintf("%s.%s", runnerAttr.Name, nameAttr.Name)
	index := -1

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

			var depNode *Node
			var nodeFound bool

			if ref.Index == -1 {
				logger.Debug("Handling shorthand implicit reference.", "instancing_mode", depStepConfig.Instancing)
				placeholderID := fmt.Sprintf("step.%s", ref.FullName)
				if pNode, pFound := graph.Nodes[placeholderID]; pFound && pNode.IsPlaceholder {
					depNode = pNode
					nodeFound = true
				} else {
					if depStepConfig.Instancing == config.ModeInstanced {
						return fmt.Errorf("ambiguous implicit dependency in '%s': expression refers to instanced step '%s' without an index", node.ID, ref.FullName)
					}
					depNodeID := fmt.Sprintf("step.%s[0]", ref.FullName)
					depNode, nodeFound = graph.Nodes[depNodeID]
				}
			} else {
				depNodeID := fmt.Sprintf("step.%s[%d]", ref.FullName, ref.Index)
				depNode, nodeFound = graph.Nodes[depNodeID]
			}

			if !nodeFound || depNode == nil {
				logger.Debug("Implicit dependency reference did not resolve to a known graph node.", "ref_full_name", ref.FullName)
				continue
			}

			if err := validateOutputReference(traversal, depNode, r); err != nil {
				return err
			}

			logger.Debug("Linking implicit dependency.", "from", node.ID, "to", depNode.ID)
			if err := graph.dag.AddEdge(depNode.ID, node.ID); err != nil {
				return fmt.Errorf("error linking implicit dependency: %w", err)
			}
			// Legacy map population removed here.
			continue
		}

		if len(traversal) >= 3 && traversal.RootName() == "resource" {
			typeAttr, typeOk := traversal[1].(hcl.TraverseAttr)
			nameAttr, nameOk := traversal[2].(hcl.TraverseAttr)
			if typeOk && nameOk {
				depID := fmt.Sprintf("resource.%s.%s", typeAttr.Name, nameAttr.Name)
				if depNode, ok := graph.Nodes[depID]; ok {
					logger.Debug("Linking implicit dependency.", "from", node.ID, "to", depID)
					if err := graph.dag.AddEdge(depNode.ID, node.ID); err != nil {
						return fmt.Errorf("error linking implicit dependency: %w", err)
					}
					// Legacy map population removed here.
				}
			}
		}
	}
	return nil
}
