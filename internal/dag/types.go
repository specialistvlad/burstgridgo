package dag

import "sync"

// Graph is a collection of nodes and their dependencies, representing a DAG.
// All operations on the graph are concurrency-safe.
type Graph struct {
	// mutex protects the nodes map during concurrent access.
	mutex sync.RWMutex
	// nodes stores all nodes in the graph, keyed by their unique ID.
	nodes map[string]*node
}

// node represents a single vertex in the graph. It is un-exported to
// enforce interaction with the graph via the public API (using string IDs),
// not by direct struct manipulation.
type node struct {
	// id is the unique identifier for the node.
	id string
	// deps holds the set of nodes that this node depends on (predecessors).
	deps map[string]*node
	// dependents holds the set of nodes that depend on this node (successors).
	dependents map[string]*node
}
