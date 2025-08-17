// Package topologystore defines the interface for storing and retrieving the
// static structure of a dependency graph.
package topologystore

import (
	"context"

	"github.com/specialistvlad/burstgridgo/internal/node"
	"github.com/specialistvlad/burstgridgo/internal/nodeid"
)

// Store is a repository for the static topology of a DAG. It is responsible
// for persisting and retrieving the nodes and their dependency edges, but not
// their dynamic run-time state.
type Store interface {
	// AddNode adds a new node to the store.
	AddNode(ctx context.Context, n *node.Node) error

	// AddDependency creates a dependency link from one node to another.
	AddDependency(ctx context.Context, from, to nodeid.Address) error

	// GetNode retrieves a single node by its address.
	GetNode(ctx context.Context, id nodeid.Address) (*node.Node, bool)

	// AllNodes returns a slice of all nodes in the topology.
	AllNodes(ctx context.Context) []*node.Node

	// DependenciesOf returns the addresses of all nodes that the given node
	// directly depends on.
	DependenciesOf(ctx context.Context, id nodeid.Address) ([]nodeid.Address, error)
}
