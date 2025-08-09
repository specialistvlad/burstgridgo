// links_explicit.go
package builder

import (
	"context"
	"fmt"

	"github.com/vk/burstgridgo/internal/config"
	"github.com/vk/burstgridgo/internal/ctxlog"
)

// linkExplicitDeps resolves dependencies from a `depends_on` block.
func linkExplicitDeps(ctx context.Context, node *Node, dependsOn []string, model *config.Model, graph *Graph) error {
	baseLogger := ctxlog.FromContext(ctx).With("node_id", node.ID)

	stepConfigMap := make(map[string]*config.Step)
	for _, step := range model.Grid.Steps {
		key := fmt.Sprintf("%s.%s", step.RunnerType, step.Name)
		stepConfigMap[key] = step
	}

	for _, depAddrRaw := range dependsOn {
		logger := baseLogger.With("depends_on", depAddrRaw)
		logger.Debug("Resolving explicit dependency.")

		parsedAddr, err := parseDepAddress(depAddrRaw)
		if err != nil {
			return err
		}
		logger.Debug("Parsed dependency address.", "name", parsedAddr.Name, "index", parsedAddr.Index)

		resourceID := "resource." + parsedAddr.Name
		if depNode, found := graph.Nodes[resourceID]; found {
			logger.Debug("Resolved as dependency on resource.", "to_node_id", depNode.ID)
			if err := graph.dag.AddEdge(depNode.ID, node.ID); err != nil {
				return fmt.Errorf("error linking explicit dependency: %w", err)
			}
			continue
		}

		depStepConfig, ok := stepConfigMap[parsedAddr.Name]
		if !ok {
			return fmt.Errorf("node '%s' depends on non-existent identifier '%s'", node.ID, depAddrRaw)
		}

		var depNode *Node
		var found bool

		if parsedAddr.Index == -1 {
			logger.Debug("Handling shorthand dependency reference.", "step_name", parsedAddr.Name)
			placeholderID := fmt.Sprintf("step.%s", parsedAddr.Name)
			if pNode, pFound := graph.Nodes[placeholderID]; pFound && pNode.IsPlaceholder {
				logger.Debug("Shorthand reference resolved to a placeholder node.", "to_node_id", pNode.ID)
				depNode = pNode
				found = true
			} else {
				if depStepConfig.Instancing == config.ModeInstanced {
					err := fmt.Errorf("ambiguous dependency in '%s': '%s' refers to an instanced step. Use index syntax (e.g., '%s[0]')", node.ID, depAddrRaw, depAddrRaw)
					logger.Error("Ambiguous dependency detected.", "error", err)
					return err
				}
				depNodeID := fmt.Sprintf("step.%s[0]", parsedAddr.Name)
				logger.Debug("Shorthand reference resolved to singular step instance.", "to_node_id", depNodeID)
				depNode, found = graph.Nodes[depNodeID]
			}
		} else {
			logger.Debug("Handling indexed dependency reference.", "step_name", parsedAddr.Name, "index", parsedAddr.Index)
			depNodeID := fmt.Sprintf("step.%s[%d]", parsedAddr.Name, parsedAddr.Index)
			depNode, found = graph.Nodes[depNodeID]
		}

		if !found || depNode == nil {
			return fmt.Errorf("node '%s' depends on non-existent identifier instance '%s'", node.ID, depAddrRaw)
		}

		logger.Debug("Linking explicit dependency.", "from_node_id", node.ID, "to_node_id", depNode.ID)
		if err := graph.dag.AddEdge(depNode.ID, node.ID); err != nil {
			return fmt.Errorf("error linking explicit dependency: %w", err)
		}
	}
	return nil
}
