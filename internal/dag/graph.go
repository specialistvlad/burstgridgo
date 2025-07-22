// ./internal/dag/graph.go

package dag

import (
	"fmt"
	"sync"

	"github.com/hashicorp/hcl/v2"
	"github.com/vk/burstgridgo/internal/engine"
	"github.com/zclconf/go-cty/cty"
)

// State represents the execution state of a node in the graph.
type State int

const (
	Pending State = iota
	Running
	Done
	Failed
)

// Node is a single node in the execution graph.
type Node struct {
	Name   string
	Module *engine.Module

	Deps       map[string]*Node // Nodes this node depends on
	Dependents map[string]*Node // Nodes that depend on this one

	State  State
	Error  error
	Output cty.Value // Stores the output of the executed module
	mu     sync.RWMutex
}

// Graph is the collection of all nodes.
type Graph struct {
	Nodes map[string]*Node
}

// detectCycles checks for circular dependencies in the graph using DFS.
func (g *Graph) detectCycles() error {
	visiting := make(map[string]bool) // Nodes currently in the recursion stack (gray set)
	visited := make(map[string]bool)  // Nodes that have been fully processed (black set)

	var visit func(node *Node) error
	visit = func(node *Node) error {
		visiting[node.Name] = true

		for _, dep := range node.Deps {
			if visiting[dep.Name] {
				// Cycle detected!
				return fmt.Errorf("cycle detected involving module '%s'", dep.Name)
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

// NewGraph builds a dependency graph from a flat list of modules.
func NewGraph(modules []*engine.Module) (*Graph, error) {
	graph := &Graph{
		Nodes: make(map[string]*Node),
	}

	// First pass: create all nodes
	for _, m := range modules {
		if _, exists := graph.Nodes[m.Name]; exists {
			return nil, fmt.Errorf("duplicate module name found: %s", m.Name)
		}
		graph.Nodes[m.Name] = &Node{
			Name:       m.Name,
			Module:     m,
			State:      Pending,
			Deps:       make(map[string]*Node),
			Dependents: make(map[string]*Node),
		}
	}

	// Second pass: link dependencies (explicit and implicit)
	for _, node := range graph.Nodes {
		// 1. Explicit dependencies from "depends_on"
		for _, depName := range node.Module.DependsOn {
			depNode, ok := graph.Nodes[depName]
			if !ok {
				return nil, fmt.Errorf("module '%s' depends on non-existent module '%s'", node.Name, depName)
			}
			node.Deps[depName] = depNode
			depNode.Dependents[node.Name] = node
		}

		// 2. Implicit dependencies from HCL variable references.
		// Use JustAttributes to analyze expressions without full schema validation.
		attrs, diags := node.Module.Body.JustAttributes()
		if diags.HasErrors() {
			return nil, diags
		}

		for _, attr := range attrs {
			vars := attr.Expr.Variables()
			for _, v := range vars {
				if len(v) > 1 && v.RootName() == "module" {
					depName := v[1].(hcl.TraverseAttr).Name
					depNode, ok := graph.Nodes[depName]
					if !ok {
						return nil, fmt.Errorf("module '%s' refers to non-existent module '%s'", node.Name, depName)
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

	// Check for cycles in the graph.
	if err := graph.detectCycles(); err != nil {
		return nil, fmt.Errorf("error validating dependency graph: %w", err)
	}

	return graph, nil
}
