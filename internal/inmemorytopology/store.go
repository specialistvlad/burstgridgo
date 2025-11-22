// Package inmemorytopology provides an ephemeral, thread-safe, in-memory
// implementation of the topologystore.Store interface.
//
// # Purpose
//
// This package implements the topology store for local execution sessions.
// It stores the static DAG structure (nodes and dependencies) in memory using
// Go maps, with a sync.RWMutex for thread-safe concurrent access.
//
// # Characteristics
//
//   - **Ephemeral:** Created fresh for each execution session, not persistent
//   - **Thread-Safe:** All methods use appropriate locking (RLock for reads, Lock for writes)
//   - **Write-Once-Read-Many:** Populated during graph construction, then read-only during execution
//   - **Fast Lookups:** O(1) node retrieval and dependency lookups using hash maps
//
// # When to Use
//
// This implementation is suitable for:
//   - Local development and testing
//   - Single-machine execution
//   - Workflows where the entire DAG fits comfortably in memory
//
// For distributed execution or workflows requiring topology persistence,
// a different implementation (e.g., backed by a distributed store) would be needed.
package inmemorytopology

import (
	"context"
	"fmt"
	"sync"

	"github.com/specialistvlad/burstgridgo/internal/node"
	"github.com/specialistvlad/burstgridgo/internal/nodeid"
	"github.com/specialistvlad/burstgridgo/internal/topologystore"
)

// Store is an in-memory implementation of topologystore.Store using Go maps
// and sync.RWMutex for thread-safe concurrent access.
//
// The store maintains two internal maps:
//   - nodes: Maps node ID strings to node.Node pointers (the DAG vertices)
//   - deps: Maps node ID strings to sets of dependency ID strings (the DAG edges)
//
// Thread-safety is guaranteed by using sync.RWMutex:
//   - Write operations (AddNode, AddDependency) use exclusive locks
//   - Read operations (GetNode, AllNodes, DependenciesOf) use shared read locks
type Store struct {
	mu    sync.RWMutex                   // Protects concurrent access to nodes and deps
	nodes map[string]*node.Node          // Map of node ID -> Node
	deps  map[string]map[string]struct{} // Map of node ID -> set of dependency IDs it depends on
}

// New creates a new, empty in-memory topology store.
func New() topologystore.Store {
	return &Store{
		nodes: make(map[string]*node.Node),
		deps:  make(map[string]map[string]struct{}),
	}
}

// AddNode adds a new node to the store.
func (s *Store) AddNode(ctx context.Context, n *node.Node) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := n.ID.String()
	if _, exists := s.nodes[key]; exists {
		// Adding the same node twice is not an error, it's idempotent.
		return nil
	}
	s.nodes[key] = n
	return nil
}

// AddDependency creates a dependency link from one node to another.
func (s *Store) AddDependency(ctx context.Context, from, to nodeid.Address) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	fromKey := from.String()
	toKey := to.String()

	if _, exists := s.nodes[fromKey]; !exists {
		return fmt.Errorf("dependency source node '%s' not found in topology", fromKey)
	}
	if _, exists := s.nodes[toKey]; !exists {
		return fmt.Errorf("dependency target node '%s' not found in topology", toKey)
	}

	if s.deps[toKey] == nil {
		s.deps[toKey] = make(map[string]struct{})
	}
	s.deps[toKey][fromKey] = struct{}{}
	return nil
}

// GetNode retrieves a single node by its address.
func (s *Store) GetNode(ctx context.Context, id nodeid.Address) (*node.Node, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	node, ok := s.nodes[id.String()]
	return node, ok
}

// AllNodes returns a slice of all nodes in the topology.
func (s *Store) AllNodes(ctx context.Context) []*node.Node {
	s.mu.RLock()
	defer s.mu.RUnlock()

	nodes := make([]*node.Node, 0, len(s.nodes))
	for _, n := range s.nodes {
		nodes = append(nodes, n)
	}
	return nodes
}

// DependenciesOf returns the addresses of all nodes that the given node depends on.
func (s *Store) DependenciesOf(ctx context.Context, id nodeid.Address) ([]nodeid.Address, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := id.String()
	if _, exists := s.nodes[key]; !exists {
		return nil, fmt.Errorf("node '%s' not found in topology", key)
	}

	depSet, ok := s.deps[key]
	if !ok {
		return []nodeid.Address{}, nil // No dependencies
	}

	deps := make([]nodeid.Address, 0, len(depSet))
	for depKey := range depSet {
		// This should not fail if our data is consistent, but defensive parsing is good.
		addr, err := nodeid.Parse(depKey)
		if err != nil {
			return nil, fmt.Errorf("internal inconsistency: failed to parse stored dependency key '%s': %w", depKey, err)
		}
		deps = append(deps, *addr)
	}
	return deps, nil
}
