// Package inmemorytopology provides a simple, thread-safe, in-memory
// implementation of the topologystore.Store interface.
package inmemorytopology

import (
	"context"
	"fmt"
	"sync"

	"github.com/specialistvlad/burstgridgo/internal/node"
	"github.com/specialistvlad/burstgridgo/internal/nodeid"
	"github.com/specialistvlad/burstgridgo/internal/topologystore"
)

// Store implements the topologystore.Store interface using maps and a mutex
// for thread-safe concurrent access.
type Store struct {
	mu    sync.RWMutex
	nodes map[string]*node.Node
	deps  map[string]map[string]struct{} // Key: node ID, Value: set of dependency IDs
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
