package dag

import (
	"context"
	"fmt"

	"github.com/vk/burstgridgo/internal/config"
	"github.com/vk/burstgridgo/internal/ctxlog"
)

// linkExplicitDeps resolves dependencies from a `depends_on` block.
func linkExplicitDeps(ctx context.Context, node *Node, dependsOn []string, model *config.Model, graph *Graph) error {
	baseLogger := ctxlog.FromContext(ctx)

	// Build a lookup map for step configs for efficient access.
	// The key is "runner_type.instance_name".
	stepConfigMap := make(map[string]*config.Step)
	for _, step := range model.Grid.Steps {
		key := fmt.Sprintf("%s.%s", step.RunnerType, step.Name)
		stepConfigMap[key] = step
	}

	for _, depAddrRaw := range dependsOn {
		logger := baseLogger.With("node_id", node.ID, "depends_on", depAddrRaw)
		logger.Debug("Resolving explicit dependency.")

		parsedAddr, err := parseDepAddress(depAddrRaw)
		if err != nil {
			return err
		}

		// First, check if it's a resource dependency. Resources are simpler as they are not instanced.
		resourceID := "resource." + parsedAddr.Name
		if depNode, found := graph.Nodes[resourceID]; found {
			logger.Debug("Resolved as dependency on resource.", "to_node_id", depNode.ID)
			node.Deps[depNode.ID] = depNode
			depNode.Dependents[node.ID] = node
			continue
		}

		// If not a resource, assume it's a step dependency.
		depStepConfig, ok := stepConfigMap[parsedAddr.Name]
		if !ok {
			return fmt.Errorf("node '%s' depends on non-existent identifier '%s'", node.ID, depAddrRaw)
		}

		var depNode *Node
		var found bool

		if parsedAddr.Index == -1 { // Shorthand reference (e.g., "http_request.my_api")
			logger.Debug("Handling shorthand dependency reference.", "step_name", parsedAddr.Name)

			// First, check if this shorthand refers to a placeholder node.
			placeholderID := fmt.Sprintf("step.%s", parsedAddr.Name)
			if pNode, pFound := graph.Nodes[placeholderID]; pFound && pNode.IsPlaceholder {
				logger.Debug("Shorthand reference resolved to a placeholder node.", "to_node_id", pNode.ID)
				depNode = pNode
				found = true
			} else {
				// If not a placeholder, apply standard instancing rules.
				if depStepConfig.Instancing == config.ModeInstanced {
					err := fmt.Errorf("ambiguous dependency in '%s': '%s' refers to an instanced step. Use index syntax (e.g., '%s[0]') to specify which instance", node.ID, depAddrRaw, depAddrRaw)
					logger.Error("Ambiguous dependency detected.", "error", err)
					return err
				}
				// It's a singular step, so it resolves to the [0] instance.
				depNodeID := fmt.Sprintf("step.%s[0]", parsedAddr.Name)
				logger.Debug("Shorthand reference resolved to singular step instance.", "to_node_id", depNodeID)
				depNode, found = graph.Nodes[depNodeID]
			}
		} else { // Indexed reference (e.g., "http_request.my_api[2]")
			logger.Debug("Handling indexed dependency reference.", "step_name", parsedAddr.Name, "index", parsedAddr.Index)
			depNodeID := fmt.Sprintf("step.%s[%d]", parsedAddr.Name, parsedAddr.Index)
			depNode, found = graph.Nodes[depNodeID]
		}

		if !found || depNode == nil {
			return fmt.Errorf("node '%s' depends on non-existent identifier instance '%s'", node.ID, depAddrRaw)
		}

		if _, exists := node.Deps[depNode.ID]; !exists {
			logger.Debug("Linking explicit dependency.", "from_node_id", node.ID, "to_node_id", depNode.ID)
			node.Deps[depNode.ID] = depNode
			depNode.Dependents[node.ID] = node
		}
	}
	return nil
}
