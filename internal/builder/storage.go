package builder

import (
	"context"
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/specialistvlad/burstgridgo/internal/config"
	"github.com/specialistvlad/burstgridgo/internal/ctxlog"
	"github.com/specialistvlad/burstgridgo/internal/dag"
	"github.com/specialistvlad/burstgridgo/internal/node"
	"github.com/specialistvlad/burstgridgo/internal/nodeid"
	"github.com/specialistvlad/burstgridgo/internal/registry"
)

// Storage is the primary artifact of the builder. It represents the complete,
// validated execution plan as a collection of Nodes and their dependencies.
type Storage struct {
	// Nodes provides a fast, ID-based lookup for any Node in the graph.
	Nodes map[string]*node.Node

	// dag holds the generic graph topology and is used for all topological
	// operations like cycle detection and dependency querying. It is unexported
	// to ensure all interactions are mediated by the builder's methods.
	dag *dag.Graph
}

// createNodes performs the first pass of graph creation, populating the graph
// with all Nodes defined in the configuration.
func (graph *Storage) createNodes(ctx context.Context, grid *config.Grid) {
	logger := ctxlog.FromContext(ctx)
	logger.Debug("Starting Node creation pass.")

	for _, s := range grid.Steps {
		expandedSteps, isPlaceholder := expandStep(s)

		if isPlaceholder {
			idStr := fmt.Sprintf("step.%s.%s", s.RunnerType, s.Name)
			logger.Debug("Creating placeholder step Node.", "id", idStr)
			if _, exists := graph.Nodes[idStr]; exists {
				logger.Warn("Duplicate step definition found, it will be overwritten.", "id", idStr)
			}
			addr, err := nodeid.Parse(idStr)
			if err != nil {
				panic(fmt.Sprintf("internal error: failed to parse generated placeholder ID %q: %v", idStr, err))
			}
			var n = node.CreateStepNode(addr, s, true)
			graph.Nodes[n.ID()] = n
			graph.dag.AddNode(n.ID())
		} else {
			logger.Debug("Creating static step Nodes.", "name", s.Name, "instance_count", len(expandedSteps))
			for i, expandedS := range expandedSteps {
				idStr := fmt.Sprintf("step.%s.%s[%d]", expandedS.RunnerType, expandedS.Name, i)
				if _, exists := graph.Nodes[idStr]; exists {
					logger.Warn("Duplicate step definition found, it will be overwritten.", "id", idStr)
				}
				addr, err := nodeid.Parse(idStr)
				if err != nil {
					panic(fmt.Sprintf("internal error: failed to parse generated static ID %q: %v", idStr, err))
				}
				var n = node.CreateStepNode(addr, s, false)
				graph.Nodes[n.ID()] = n
				graph.dag.AddNode(n.ID())
			}
		}
	}
	for _, r := range grid.Resources {
		idStr := fmt.Sprintf("resource.%s.%s", r.AssetType, r.Name)
		logger.Debug("Creating resource Node.", "id", idStr)
		if _, exists := graph.Nodes[idStr]; exists {
			logger.Warn("Duplicate resource definition found, it will be overwritten.", "id", idStr)
		}
		addr, err := nodeid.Parse(idStr)
		if err != nil {
			panic(fmt.Sprintf("internal error: failed to parse generated resource ID %q: %v", idStr, err))
		}
		var n = node.CreateResourceNode(addr, r)
		graph.Nodes[n.ID()] = n
		graph.dag.AddNode(n.ID())
	}
	logger.Debug("Finished Node creation pass.")
}

// linkNodes performs the second pass, establishing dependency edges between Nodes.
func (graph *Storage) linkNodes(ctx context.Context, model *config.Model, r *registry.Registry) error {
	logger := ctxlog.FromContext(ctx)
	logger.Debug("Starting Node linking pass.")

	for _, Node := range graph.Nodes {
		NodeLogger := logger.With("Node_id", Node.ID())
		NodeLogger.Debug("Processing dependencies for Node.")
		var dependsOn []string
		var expressions []hcl.Expression

		if Node.Type == node.StepNode {
			if Node.IsPlaceholder && Node.StepConfig.Count != nil {
				expressions = append(expressions, Node.StepConfig.Count)
			}
			dependsOn = Node.StepConfig.DependsOn
			for _, expr := range Node.StepConfig.Arguments {
				expressions = append(expressions, expr)
			}
			for _, expr := range Node.StepConfig.Uses {
				expressions = append(expressions, expr)
			}
		} else { // ResourceNode
			dependsOn = Node.ResourceConfig.DependsOn
			for _, expr := range Node.ResourceConfig.Arguments {
				expressions = append(expressions, expr)
			}
		}

		if len(dependsOn) > 0 {
			NodeLogger.Debug("Linking explicit dependencies.", "count", len(dependsOn))
			if err := graph.linkExplicitDeps(ctx, Node, dependsOn, model); err != nil {
				return err
			}
		}

		if len(expressions) > 0 {
			NodeLogger.Debug("Linking implicit dependencies from expressions.", "count", len(expressions))
			for _, expr := range expressions {
				if err := graph.linkImplicitDeps(ctx, Node, expr, model, r); err != nil {
					return err
				}
			}
		}
	}
	logger.Debug("Finished Node linking pass.")
	return nil
}

// Dependencies returns a slice of Nodes that the given Node directly depends on.
// It queries the underlying generic DAG and converts the returned string IDs into
// rich *node.Node pointers used by the application.
func (g *Storage) Dependencies(NodeID string) ([]*node.Node, error) {
	depIDs, err := g.dag.Dependencies(NodeID)
	if err != nil {
		return nil, err
	}
	deps := make([]*node.Node, 0, len(depIDs))
	for _, id := range depIDs {
		// This lookup is safe because the graph is static after being built.
		if Node, ok := g.Nodes[id]; ok {
			deps = append(deps, Node)
		}
	}
	return deps, nil
}

// Dependents returns a slice of Nodes that directly depend on the given Node.
// It queries the underlying generic DAG and converts the returned string IDs into
// rich *node.Node pointers used by the application.
func (g *Storage) Dependents(NodeID string) ([]*node.Node, error) {
	depIDs, err := g.dag.Dependents(NodeID)
	if err != nil {
		return nil, err
	}
	deps := make([]*node.Node, 0, len(depIDs))
	for _, id := range depIDs {
		// This lookup is safe because the graph is static after being built.
		if Node, ok := g.Nodes[id]; ok {
			deps = append(deps, Node)
		}
	}
	return deps, nil
}

// SetInitialCounters prepares a node for the executor by setting its atomic
// counters based on the final graph topology.
// It prevents code from refactoring. Must be moved to graph struct.
func (g *Storage) SetInitialCounters(ctx context.Context, n *node.Node) error {
	logger := ctxlog.FromContext(ctx).With("node_id", n.ID())

	deps, err := g.Dependencies(n.ID())
	if err != nil {
		return err
	}
	depCount := int32(len(deps))
	n.SetDepCount(depCount)
	logger.Debug("Initialized dependency counter.", "count", depCount)

	if n.Type == node.ResourceNode {
		dependents, err := g.Dependents(n.ID())
		if err != nil {
			return err
		}
		var directStepDependents int32 = 0
		for _, dependent := range dependents {
			if dependent.Type == node.StepNode {
				directStepDependents++
			}
		}
		n.SetDescCount(directStepDependents)
		logger.Debug("Initialized resource descendant counter.", "count", directStepDependents)
	}
	return nil
}
