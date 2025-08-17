package graph

import (
	"context"

	"github.com/specialistvlad/burstgridgo/internal/node"
	"github.com/specialistvlad/burstgridgo/internal/nodeid"
)

// Graph represents the live, stateful dependency graph for a single execution
// run. It provides a unified, thread-safe interface for querying the graph's
// topology and updating the state of its nodes. It is the single source of
// truth for a run's progress, composing lower-level stores.
type Graph interface {
	// Node retrieves a node by its structured address.
	Node(ctx context.Context, id nodeid.Address) (*node.Node, bool)

	// DependenciesOf retrieves the direct dependencies for a given node.
	DependenciesOf(ctx context.Context, id nodeid.Address) ([]*node.Node, error)

	// NodeStatus retrieves the current execution status of a given node.
	NodeStatus(ctx context.Context, id nodeid.Address) (node.Status, bool)

	// AllNodes returns a slice of all nodes in the graph.
	AllNodes(ctx context.Context) []*node.Node

	// MarkRunning sets a node's status to 'running'.
	MarkRunning(ctx context.Context, id nodeid.Address) error

	// MarkCompleted sets a node's status to 'completed' and records its output.
	MarkCompleted(ctx context.Context, id nodeid.Address, output any) error

	// MarkFailed sets a node's status to 'failed' and records the error.
	MarkFailed(ctx context.Context, id nodeid.Address, nodeErr error) error

	// MarkSkipped sets a node's status to 'skipped'.
	MarkSkipped(ctx context.Context, id nodeid.Address) error
}
