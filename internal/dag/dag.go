package dag

import (
	"fmt"
)

// New creates and returns an initialized, empty Graph.
func New() *Graph {
	return &Graph{
		nodes: make(map[string]*node),
	}
}

// AddNode adds a new node with the given ID to the graph. If a node with
// the same ID already exists, the function does nothing.
func (g *Graph) AddNode(id string) {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	if _, ok := g.nodes[id]; ok {
		return
	}

	g.nodes[id] = &node{
		id:         id,
		deps:       make(map[string]*node),
		dependents: make(map[string]*node),
	}
}

// AddEdge creates a directed edge from the `fromID` node to the `toID` node.
// This signifies that `toID` has a dependency on `fromID`. An error is returned
// if either node does not exist or if the edge would create a self-reference.
func (g *Graph) AddEdge(fromID, toID string) error {
	if fromID == toID {
		return fmt.Errorf("self-referential edge not allowed: %s -> %s", fromID, fromID)
	}

	g.mutex.Lock()
	defer g.mutex.Unlock()

	fromNode, ok := g.nodes[fromID]
	if !ok {
		return fmt.Errorf("source node not found: %s", fromID)
	}

	toNode, ok := g.nodes[toID]
	if !ok {
		return fmt.Errorf("destination node not found: %s", toID)
	}

	toNode.deps[fromID] = fromNode
	fromNode.dependents[toID] = toNode

	return nil
}

// NEW METHOD
// Dependencies returns a slice of node IDs that the given node depends on.
func (g *Graph) Dependencies(id string) ([]string, error) {
	g.mutex.RLock()
	defer g.mutex.RUnlock()

	n, ok := g.nodes[id]
	if !ok {
		return nil, fmt.Errorf("node not found: %s", id)
	}

	deps := make([]string, 0, len(n.deps))
	for depID := range n.deps {
		deps = append(deps, depID)
	}
	return deps, nil
}

// NEW METHOD
// Dependents returns a slice of node IDs that depend on the given node.
func (g *Graph) Dependents(id string) ([]string, error) {
	g.mutex.RLock()
	defer g.mutex.RUnlock()

	n, ok := g.nodes[id]
	if !ok {
		return nil, fmt.Errorf("node not found: %s", id)
	}

	dependents := make([]string, 0, len(n.dependents))
	for depID := range n.dependents {
		dependents = append(dependents, depID)
	}
	return dependents, nil
}

// DetectCycles checks the graph for any cycles. It returns a non-nil error
// if a cycle is found, indicating the first node involved in the detected cycle.
func (g *Graph) DetectCycles() error {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	// Use classic depth-first search with three sets of nodes:
	// permanent: nodes that have been fully visited and are not part of a cycle.
	// temporary: nodes currently in the recursion stack for the current traversal.
	// unvisited: all other nodes.
	permanent := make(map[string]bool)
	temporary := make(map[string]bool)

	var visit func(n *node) error
	visit = func(n *node) error {
		if permanent[n.id] {
			return nil // Already visited and known to be safe.
		}
		if temporary[n.id] {
			// We've hit a node that's already in our recursion stack, so we have a cycle.
			return fmt.Errorf("cycle detected involving node '%s'", n.id)
		}

		temporary[n.id] = true

		for _, dependent := range n.dependents {
			if err := visit(dependent); err != nil {
				return err // Propagate the error up.
			}
		}

		// All dependents have been visited, so we can move this node from temporary to permanent.
		delete(temporary, n.id)
		permanent[n.id] = true

		return nil
	}

	// Visit every node in the graph.
	for _, n := range g.nodes {
		if !permanent[n.id] {
			if err := visit(n); err != nil {
				return err
			}
		}
	}

	return nil
}
