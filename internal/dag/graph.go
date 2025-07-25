package dag

import (
	"fmt"
	"sync/atomic"

	"github.com/hashicorp/hcl/v2"
	"github.com/vk/burstgridgo/internal/engine"
)

// NodeType distinguishes between node types in the graph.
type NodeType int

const (
	StepNode NodeType = iota
	ResourceNode
)

// State represents the execution state of a node in the graph.
type State int32

const (
	Pending State = iota
	Running
	Done
	Failed
)

// Node is a single node in the execution graph.
type Node struct {
	ID              string // Unique ID, e.g., "step.http_request.my_step"
	Name            string // The instance name from HCL
	Type            NodeType
	StepConfig      *engine.Step
	ResourceConfig  *engine.Resource
	Deps            map[string]*Node
	Dependents      map[string]*Node
	depCount        atomic.Int32
	descendantCount atomic.Int32 // For resources: counts steps that depend on it.
	State           atomic.Int32
	Error           error
	Output          any // For steps: cty.Value; for resources: live Go object
}

// Graph is the collection of all nodes.
type Graph struct {
	Nodes map[string]*Node
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

// NewGraph builds a dependency graph from a user's grid configuration.
func NewGraph(config *engine.GridConfig) (*Graph, error) {
	graph := &Graph{Nodes: make(map[string]*Node)}

	// First pass: create all nodes for steps and resources.
	for _, s := range config.Steps {
		id := fmt.Sprintf("step.%s.%s", s.RunnerType, s.Name)
		if _, exists := graph.Nodes[id]; exists {
			return nil, fmt.Errorf("duplicate step found: %s", id)
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
			return nil, fmt.Errorf("duplicate resource found: %s", id)
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

	// Second pass: link dependencies (explicit, implicit, and resource uses).
	for _, node := range graph.Nodes {
		// 1. Explicit `depends_on`
		var dependsOn []string
		if node.Type == StepNode {
			dependsOn = node.StepConfig.DependsOn
		} else { // ResourceNode
			dependsOn = node.ResourceConfig.DependsOn
		}
		if err := linkExplicitDeps(node, dependsOn, graph); err != nil {
			return nil, err
		}

		// 2. Implicit dependencies for steps (from `arguments` and `uses` blocks)
		if node.Type == StepNode {
			if node.StepConfig.Arguments != nil && node.StepConfig.Arguments.Body != nil {
				if err := linkImplicitDeps(node, node.StepConfig.Arguments.Body, graph); err != nil {
					return nil, err
				}
			}
			if node.StepConfig.Uses != nil && node.StepConfig.Uses.Body != nil {
				if err := linkImplicitDeps(node, node.StepConfig.Uses.Body, graph); err != nil {
					return nil, err
				}
			}
		}
	}

	// Third pass: initialize dependency and descendant counters.
	for _, node := range graph.Nodes {
		node.depCount.Store(int32(len(node.Deps)))
		if node.Type == ResourceNode {
			descendants := make(map[string]struct{})
			countDescendants(node, descendants)
			node.descendantCount.Store(int32(len(descendants)))
		}
	}

	if err := graph.detectCycles(); err != nil {
		return nil, fmt.Errorf("error validating dependency graph: %w", err)
	}

	return graph, nil
}

// linkExplicitDeps resolves dependencies listed in a `depends_on` block.
func linkExplicitDeps(node *Node, dependsOn []string, graph *Graph) error {
	for _, depAddr := range dependsOn {
		// Try resolving as a step first, then as a resource.
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
			node.Deps[depNode.ID] = depNode
			depNode.Dependents[node.ID] = node
		}
	}
	return nil
}

// countDescendants performs a traversal to find all unique steps that depend on a resource.
func countDescendants(node *Node, visited map[string]struct{}) {
	for _, dependent := range node.Dependents {
		if dependent.Type == StepNode {
			if _, exists := visited[dependent.ID]; !exists {
				visited[dependent.ID] = struct{}{}
				countDescendants(dependent, visited)
			}
		}
	}
}

// linkImplicitDeps parses a body for variable traversals and links the node to its dependencies.
func linkImplicitDeps(node *Node, body hcl.Body, graph *Graph) error {
	attrs, diags := body.JustAttributes()
	if diags.HasErrors() {
		return diags
	}

	for _, attr := range attrs {
		for _, traversal := range attr.Expr.Variables() {
			// We're looking for traversals that start with 'step' or 'resource'
			// and have at least two more parts (type and name).
			if len(traversal) < 3 {
				continue
			}
			rootName := traversal.RootName()
			if rootName != "step" && rootName != "resource" {
				continue
			}

			// The next two parts must be attribute traversals for type and name.
			typeAttr, typeOk := traversal[1].(hcl.TraverseAttr)
			nameAttr, nameOk := traversal[2].(hcl.TraverseAttr)
			if !typeOk || !nameOk {
				continue
			}

			// Reconstruct the dependency ID from the first three parts of the traversal.
			depID := fmt.Sprintf("%s.%s.%s", rootName, typeAttr.Name, nameAttr.Name)

			depNode, ok := graph.Nodes[depID]
			if !ok {
				// This is not a reference to a known node, so we can ignore it.
				continue
			}

			// Link the dependency if it's not already linked.
			if _, exists := node.Deps[depID]; !exists {
				node.Deps[depID] = depNode
				depNode.Dependents[node.ID] = node
			}
		}
	}
	return nil
}
