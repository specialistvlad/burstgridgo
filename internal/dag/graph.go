package dag

import (
	"fmt"
	"sync/atomic"

	"github.com/hashicorp/hcl/v2"
	"github.com/vk/burstgridgo/internal/engine"
	"github.com/zclconf/go-cty/cty"
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
	Name       string
	Step       *engine.Step // <-- Changed from Module
	Deps       map[string]*Node
	Dependents map[string]*Node
	depCount   atomic.Int32
	State      atomic.Int32
	Error      error
	Output     cty.Value
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
		visiting[node.Name] = true

		for _, dep := range node.Deps {
			if visiting[dep.Name] {
				return fmt.Errorf("cycle detected involving step '%s'", dep.Name)
			}
			if !visited[dep.Name] {
				if err := visit(dep); err != nil {
					return err
				}
			}
		}

		delete(visiting, node.Name)
		visited[node.Name] = true
		return nil
	}

	for _, node := range g.Nodes {
		if !visited[node.Name] {
			if err := visit(node); err != nil {
				return err
			}
		}
	}

	return nil
}

// NewGraph builds a dependency graph from a flat list of steps.
func NewGraph(steps []*engine.Step) (*Graph, error) {
	graph := &Graph{
		Nodes: make(map[string]*Node),
	}

	// First pass: create all nodes
	for _, s := range steps {
		if _, exists := graph.Nodes[s.Name]; exists {
			return nil, fmt.Errorf("duplicate step name found: %s", s.Name)
		}
		graph.Nodes[s.Name] = &Node{
			Name:       s.Name,
			Step:       s,
			State:      atomic.Int32{},
			depCount:   atomic.Int32{},
			Deps:       make(map[string]*Node),
			Dependents: make(map[string]*Node),
		}
	}

	// Second pass: link dependencies (explicit and implicit)
	for _, node := range graph.Nodes {
		// 1. Explicit dependencies from "depends_on"
		for _, depName := range node.Step.DependsOn {
			depNode, ok := graph.Nodes[depName]
			if !ok {
				return nil, fmt.Errorf("step '%s' depends on non-existent step '%s'", node.Name, depName)
			}
			node.Deps[depName] = depNode
			depNode.Dependents[node.Name] = node
		}

		// 2. Implicit dependencies from HCL variable references.
		attrs, diags := node.Step.Arguments.JustAttributes()
		if diags.HasErrors() {
			return nil, diags
		}

		for _, attr := range attrs {
			vars := attr.Expr.Variables()
			for _, v := range vars {
				// This logic needs to be updated to look for `step.` instead of `module.`
				// For now, let's assume it still works with a `step` variable.
				if len(v) > 1 && v.RootName() == "step" { // <-- Changed from "module"
					depName := v[1].(hcl.TraverseAttr).Name
					depNode, ok := graph.Nodes[depName]
					if !ok {
						return nil, fmt.Errorf("step '%s' refers to non-existent step '%s'", node.Name, depName)
					}
					// Link them if not already linked
					if _, exists := node.Deps[depName]; !exists {
						node.Deps[depName] = depNode
						depNode.Dependents[node.Name] = node
					}
				}
			}
		}
	}

	// Third pass: initialize dependency counters for all nodes.
	for _, node := range graph.Nodes {
		node.depCount.Store(int32(len(node.Deps)))
	}

	// Check for cycles in the graph.
	if err := graph.detectCycles(); err != nil {
		return nil, fmt.Errorf("error validating dependency graph: %w", err)
	}

	return graph, nil
}
