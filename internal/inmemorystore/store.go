// Package inmemorystore provides an ephemeral, thread-safe, in-memory
// implementation of the nodestore.Store interface.
//
// # Purpose
//
// This package implements the node state store for local execution sessions.
// It stores mutable execution state (status, outputs, errors) in memory using
// sync.Map for fine-grained concurrent access without global lock contention.
//
// # Characteristics
//
//   - **Ephemeral:** Created fresh for each execution session, not persistent
//   - **Thread-Safe:** Uses sync.Map for lock-free concurrent access in most cases
//   - **High-Write:** Optimized for frequent state updates during parallel execution
//   - **Fast Lookups:** O(1) average case for status/output/error retrieval
//
// # Concurrency Model
//
// Unlike inmemorytopology which uses RWMutex, this store uses sync.Map because:
//   - **Write-Heavy Workload:** Executor constantly updates node status/output/error
//   - **Independent Keys:** Each node's state is independent, enabling fine-grained locking
//   - **Concurrent Reads + Writes:** Builder reads outputs while executor writes statuses
//
// sync.Map is optimized for this pattern where the key space is relatively stable
// (all nodes known upfront) but values change frequently.
//
// # When to Use
//
// This implementation is suitable for:
//   - Local development and testing
//   - Single-machine execution
//   - Workflows where all node state fits comfortably in memory
//
// For distributed execution or workflows requiring state persistence/recovery,
// a different implementation (e.g., backed by Redis, etcd) would be needed.
package inmemorystore

import (
	"context"
	"sync"

	"github.com/specialistvlad/burstgridgo/internal/node"
	"github.com/specialistvlad/burstgridgo/internal/nodeid"
	"github.com/specialistvlad/burstgridgo/internal/nodestore"
)

// Store is an in-memory implementation of nodestore.Store using sync.Map
// for fine-grained concurrent access without global lock contention.
//
// The store maintains three independent sync.Maps:
//   - states: Maps node ID strings to node.Status (Pending, Running, Completed, Failed)
//   - outputs: Maps node ID strings to execution outputs (any type, typically map[string]interface{})
//   - errors: Maps node ID strings to error objects for failed nodes
//
// Thread-safety is guaranteed by sync.Map's built-in concurrency control:
//   - Multiple goroutines can safely read/write different keys simultaneously
//   - Read-heavy workloads (after initial writes) are lock-free
//   - Write contention on the same key is handled internally by sync.Map
type Store struct {
	states  sync.Map // Key: node ID string, Value: node.Status
	outputs sync.Map // Key: node ID string, Value: any (output data)
	errors  sync.Map // Key: node ID string, Value: error
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
