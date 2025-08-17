// Package inmemorystore provides a simple, thread-safe, in-memory
// implementation of the nodestore.Store interface.
package inmemorystore

import (
	"context"
	"sync"

	"github.com/specialistvlad/burstgridgo/internal/node"
	"github.com/specialistvlad/burstgridgo/internal/nodeid"
	"github.com/specialistvlad/burstgridgo/internal/nodestore"
)

// Store implements the nodestore.Store interface using sync.Map for
// fine-grained, concurrent access without global lock contention.
type Store struct {
	// Each map stores a different aspect of the node's state.
	// The key is always the nodeid.Address.String().
	states  sync.Map // Stores node.Status
	outputs sync.Map // Stores any
	errors  sync.Map // Stores error
}

// New creates a new, empty in-memory node state store.
func New() nodestore.Store {
	return &Store{}
}

// SetStatus updates the execution status of a specific node.
func (s *Store) SetStatus(ctx context.Context, id nodeid.Address, status node.Status) error {
	s.states.Store(id.String(), status)
	return nil
}

// GetStatus retrieves the execution status of a specific node.
// If a status has not been set, it returns StatusPending.
func (s *Store) GetStatus(ctx context.Context, id nodeid.Address) (node.Status, error) {
	status, ok := s.states.Load(id.String())
	if !ok {
		return node.StatusPending, nil
	}
	return status.(node.Status), nil
}

// SetOutput records the successful output of a node.
func (s *Store) SetOutput(ctx context.Context, id nodeid.Address, output any) error {
	s.outputs.Store(id.String(), output)
	return nil
}

// GetOutput retrieves the recorded output of a completed node.
func (s *Store) GetOutput(ctx context.Context, id nodeid.Address) (any, error) {
	output, ok := s.outputs.Load(id.String())
	if !ok {
		return nil, nil // If not found, the output is nil.
	}
	return output, nil
}

// SetError records the failure error of a node.
func (s *Store) SetError(ctx context.Context, id nodeid.Address, nodeErr error) error {
	s.errors.Store(id.String(), nodeErr)
	return nil
}

// GetError retrieves the recorded error of a failed node.
func (s *Store) GetError(ctx context.Context, id nodeid.Address) (error, error) {
	err, ok := s.errors.Load(id.String())
	if !ok {
		return nil, nil // If not found, there is no error.
	}
	return err.(error), nil
}
