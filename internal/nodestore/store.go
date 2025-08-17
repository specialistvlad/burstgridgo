// Package nodestore defines the interface for storing and retrieving the
// dynamic, run-time state of nodes in a graph.
package nodestore

import (
	"context"

	"github.com/specialistvlad/burstgridgo/internal/node"
	"github.com/specialistvlad/burstgridgo/internal/nodeid"
)

// Store is a repository for the dynamic state of nodes during a run. It does
// not know about the graph's topology, only about individual node states.
type Store interface {
	// SetStatus updates the execution status of a specific node.
	SetStatus(ctx context.Context, id nodeid.Address, status node.Status) error

	// GetStatus retrieves the execution status of a specific node.
	GetStatus(ctx context.Context, id nodeid.Address) (node.Status, error)

	// SetOutput records the successful output of a node.
	SetOutput(ctx context.Context, id nodeid.Address, output any) error

	// GetOutput retrieves the recorded output of a completed node.
	GetOutput(ctx context.Context, id nodeid.Address) (any, error)

	// SetError records the failure error of a node.
	SetError(ctx context.Context, id nodeid.Address, nodeErr error) error

	// GetError retrieves the recorded error of a failed node.
	GetError(ctx context.Context, id nodeid.Address) (error, error)
}
