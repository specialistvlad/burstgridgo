// Package topologystore defines the interface for storing and retrieving the
// static structure of a dependency graph (DAG).
//
// # Why Topology Store Exists
//
// The topology store implements a critical separation of concerns in BurstGridGo:
// it isolates the **immutable DAG structure** (nodes and their dependency relationships)
// from the **mutable execution state** (status, outputs, errors) managed by nodestore.
//
// This separation provides several architectural benefits:
//   - **Clarity:** Graph structure queries (scheduler) don't mix with state updates (executor)
//   - **Thread-Safety:** Read-heavy topology queries can use RLocks without contention from frequent state writes
//   - **Testability:** DAG structure can be validated independently of execution state
//   - **Flexibility:** Different storage backends can be swapped (in-memory, distributed, persistent)
//
// # Lifecycle and Usage
//
// The topology store is:
//   1. **Created** once per execution session (ephemeral, not persistent across runs)
//   2. **Populated** during graph construction phase (nodes + dependencies added)
//   3. **Read-only** during execution phase (scheduler queries dependencies, executor looks up nodes)
//   4. **Discarded** when the session ends
//
// The topology is write-once-read-many after the graph construction phase completes.
// During execution, the scheduler continuously queries DependenciesOf() to determine
// which nodes are ready to run, while the executor calls GetNode() to retrieve node
// configurations for task building.
package topologystore

import (
	"context"

	"github.com/specialistvlad/burstgridgo/internal/node"
	"github.com/specialistvlad/burstgridgo/internal/nodeid"
)

// Store is the interface for managing the static topology of a directed acyclic graph (DAG).
//
// The topology store is responsible for storing and retrieving:
//   - **Nodes**: The vertices in the DAG (steps, resources, etc.)
//   - **Dependencies**: The directed edges between nodes (who depends on whom)
//
// This interface does NOT manage dynamic execution state (node status, outputs, errors).
// That responsibility belongs to nodestore.Store.
//
// # Thread-Safety Requirements
//
// Implementations MUST be thread-safe for concurrent reads and writes, as the topology
// is constructed during graph building and queried heavily during parallel execution.
//
// # Typical Implementation
//
// See internal/inmemorytopology for the reference in-memory implementation using
// maps and sync.RWMutex for thread-safe concurrent access.
type Store interface {
	// AddNode registers a new node in the topology.
	//
	// This method is called during the graph construction phase to populate the DAG
	// structure. Each node represents a unit of work (step, resource, etc.) and
	// contains its configuration but NOT its execution state.
	//
	// Adding the same node twice (by ID) should be idempotent and not return an error.
	//
	// Thread-safety: Must be safe to call concurrently with other AddNode calls.
	AddNode(ctx context.Context, n *node.Node) error

	// AddDependency creates a directed dependency edge in the topology.
	//
	// This establishes that the node identified by 'to' depends on the node identified
	// by 'from', meaning 'from' must complete successfully before 'to' can start.
	//
	// Both nodes must already exist in the topology (added via AddNode) before calling
	// this method. If either node is missing, implementations should return an error.
	//
	// Example: AddDependency(ctx, "step.first", "step.second") means step.second
	// depends on step.first (step.first must run first).
	//
	// Thread-safety: Must be safe to call concurrently with other AddDependency calls.
	AddDependency(ctx context.Context, from, to nodeid.Address) error

	// GetNode retrieves a single node by its address.
	//
	// This is used during execution to look up node configurations when building tasks.
	// Returns the node and true if found, or nil and false if the node doesn't exist.
	//
	// Thread-safety: Must be safe to call concurrently with writes and other reads.
	GetNode(ctx context.Context, id nodeid.Address) (*node.Node, bool)

	// AllNodes returns all nodes currently registered in the topology.
	//
	// This is primarily used for graph analysis, debugging, and validation. The order
	// of nodes in the returned slice is not guaranteed to be deterministic unless
	// specified by the implementation.
	//
	// Thread-safety: Must be safe to call concurrently with writes and other reads.
	// The returned slice should be a snapshot and safe for the caller to iterate.
	AllNodes(ctx context.Context) []*node.Node

	// DependenciesOf returns all nodes that the given node directly depends on.
	//
	// This is the core method used by the scheduler to determine execution order.
	// The scheduler repeatedly calls this to find nodes with zero unsatisfied dependencies
	// (ready to execute).
	//
	// Returns:
	//   - A slice of node addresses that 'id' depends on (empty slice if no dependencies)
	//   - An error if the node 'id' doesn't exist in the topology
	//
	// Example: If step.third depends on [step.first, step.second], then
	// DependenciesOf("step.third") returns ["step.first", "step.second"].
	//
	// Thread-safety: Must be safe to call concurrently, as the scheduler queries this
	// heavily during parallel execution.
	DependenciesOf(ctx context.Context, id nodeid.Address) ([]nodeid.Address, error)
}
