// Package nodestore defines the interface for storing and retrieving the
// dynamic, mutable execution state of nodes during workflow execution.
//
// # Why Node Store Exists
//
// The node store implements a critical separation of concerns in BurstGridGo:
// it isolates **mutable execution state** (status, outputs, errors) from the
// **immutable DAG structure** (nodes, dependencies) managed by topologystore.
//
// This separation provides several architectural benefits:
//   - **Clarity:** State updates (executor) don't interfere with structure queries (scheduler)
//   - **Concurrency:** Frequent state writes don't block topology reads using different locks
//   - **Testability:** Execution state can be validated independently of DAG structure
//   - **Flexibility:** Different storage backends can be swapped (in-memory, distributed, persistent)
//
// # Lifecycle and Usage
//
// The node store is:
//   1. **Created** once per execution session (ephemeral, not persistent across runs)
//   2. **Initialized** with all nodes in Pending status before execution starts
//   3. **Mutated** continuously during execution as nodes transition through states
//   4. **Queried** by builder to resolve expressions referencing other node outputs
//   5. **Discarded** when the session ends
//
// During execution:
//   - **Executor** calls SetStatus/SetOutput/SetError as nodes execute
//   - **Builder** calls GetOutput to resolve expressions like `step.first.output.value`
//   - **Scheduler** queries the store (via graph) to track which nodes have completed
//
// # State Transitions
//
// Nodes follow this lifecycle:
//   Pending → Running → Completed (with output) OR Failed (with error)
package nodestore

import (
	"context"

	"github.com/specialistvlad/burstgridgo/internal/node"
	"github.com/specialistvlad/burstgridgo/internal/nodeid"
)

// Store is the interface for managing the mutable execution state of nodes during workflow execution.
//
// The node store is responsible for tracking:
//   - **Status**: Current execution state (Pending, Running, Completed, Failed)
//   - **Output**: Successful execution results (any type, stored as interface{})
//   - **Error**: Failure information for debugging and error handling
//
// This interface does NOT manage static DAG structure (nodes, dependencies).
// That responsibility belongs to topologystore.Store.
//
// # Thread-Safety Requirements
//
// Implementations MUST be thread-safe for concurrent reads and writes, as multiple
// goroutines execute nodes in parallel and simultaneously update/query state.
//
// # Typical Implementation
//
// See internal/inmemorystore for the reference in-memory implementation using
// sync.Map for fine-grained concurrent access without global lock contention.
type Store interface {
	// SetStatus updates the execution status of a node.
	//
	// This is called by the executor to track node lifecycle transitions:
	//   - Pending → Running (when execution starts)
	//   - Running → Completed (when execution succeeds)
	//   - Running → Failed (when execution errors)
	//
	// The node must already exist in the topology. This method only updates state,
	// it doesn't validate topology membership.
	//
	// Thread-safety: Must be safe to call concurrently for different nodes.
	SetStatus(ctx context.Context, id nodeid.Address, status node.Status) error

	// GetStatus retrieves the current execution status of a node.
	//
	// Used by the scheduler (via graph) to determine which nodes have completed
	// and which dependencies are satisfied.
	//
	// Returns StatusPending if no status has been set for this node yet.
	//
	// Thread-safety: Must be safe to call concurrently with SetStatus calls.
	GetStatus(ctx context.Context, id nodeid.Address) (node.Status, error)

	// SetOutput records the successful execution output of a node.
	//
	// Called by the executor after a node completes successfully. The output
	// can be any type (interface{}), typically a map[string]interface{} or struct
	// representing the node's execution results.
	//
	// This output is later retrieved by the builder when resolving expressions
	// like `step.http_request.first.output.status_code`.
	//
	// Thread-safety: Must be safe to call concurrently for different nodes.
	SetOutput(ctx context.Context, id nodeid.Address, output any) error

	// GetOutput retrieves the recorded output of a completed node.
	//
	// This is the core method used by the builder during expression resolution.
	// When evaluating `step.first.output.value`, the builder calls GetOutput("step.first")
	// and then extracts the "value" field.
	//
	// Returns nil if the node hasn't completed yet or produced no output.
	//
	// Thread-safety: Must be safe to call concurrently with SetOutput calls.
	GetOutput(ctx context.Context, id nodeid.Address) (any, error)

	// SetError records the failure error of a node.
	//
	// Called by the executor when a node's execution fails. The error contains
	// information about what went wrong and is used for debugging and error reporting.
	//
	// Thread-safety: Must be safe to call concurrently for different nodes.
	SetError(ctx context.Context, id nodeid.Address, nodeErr error) error

	// GetError retrieves the recorded error of a failed node.
	//
	// Used for debugging, error reporting, and determining why a workflow failed.
	//
	// Returns nil if the node succeeded or hasn't executed yet.
	//
	// Thread-safety: Must be safe to call concurrently with SetError calls.
	GetError(ctx context.Context, id nodeid.Address) (error, error)
}
