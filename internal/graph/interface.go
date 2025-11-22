// Package graph provides a unified, high-level interface for managing the execution graph.
//
// # Why Graph Package Exists
//
// The Graph interface serves as a facade that combines topology (structure) and node state (execution)
// into a single, cohesive API. This simplifies interactions for the executor and scheduler, which
// would otherwise need to coordinate between topologystore and nodestore directly.
//
// This separation provides several architectural benefits:
//   - **Unified API:** Executor and scheduler interact with one interface instead of two stores
//   - **Encapsulation:** Graph hides the dual-store implementation detail from consumers
//   - **Flexibility:** Future implementations could add caching, validation, or event hooks
//   - **Clarity:** Business logic (executor) doesn't know about storage implementation details
//
// # Responsibilities
//
// The graph package orchestrates two underlying stores:
//   - **Topology Store** (topologystore.Store): Provides static DAG structure (nodes, dependencies)
//   - **Node Store** (nodestore.Store): Manages mutable execution state (status, outputs, errors)
//
// The Graph interface delegates queries to the appropriate store and provides convenience
// methods that combine information from both stores (e.g., DependenciesOf returns full nodes,
// not just IDs).
//
// # Lifecycle
//
// 1. **Created** by session factory with topology and node stores injected
// 2. **Populated** during graph construction (nodes added to topology)
// 3. **Queried** during execution (scheduler finds ready nodes, executor updates state)
// 4. **Discarded** when session ends
package graph

import (
	"context"

	"github.com/specialistvlad/burstgridgo/internal/node"
	"github.com/specialistvlad/burstgridgo/internal/nodeid"
)

// Graph is a unified interface for interacting with the execution DAG, combining
// static topology queries with dynamic state updates.
//
// The graph represents the complete, stateful execution context for a single workflow run.
// It is the single source of truth for:
//   - **Structure**: Which nodes exist and how they depend on each other (via topology store)
//   - **State**: What's the current execution status of each node (via node store)
//
// # Usage Patterns
//
// **Scheduler** uses Graph to:
//   - Query all nodes: AllNodes()
//   - Find dependencies: DependenciesOf()
//   - Check node status: NodeStatus()
//
// **Executor** uses Graph to:
//   - Look up nodes for execution: Node()
//   - Update execution state: MarkRunning(), MarkCompleted(), MarkFailed()
//
// **Builder** uses Graph to:
//   - Retrieve dependency outputs for expression resolution (via node store internally)
//
// # Thread-Safety
//
// Implementations MUST be thread-safe, as multiple goroutines execute nodes in parallel
// and simultaneously query/update the graph.
//
// # Typical Implementation
//
// See internal/graph.Manager for the reference implementation that composes
// topologystore.Store and nodestore.Store.
type Graph interface {
	// Node retrieves a node's configuration by its address.
	//
	// Returns the node and true if found, or nil and false if the node doesn't exist
	// in the topology.
	//
	// Used by executor and builder to look up node configuration before execution
	// and expression resolution.
	//
	// Thread-safety: Must be safe to call concurrently.
	Node(ctx context.Context, id nodeid.Address) (*node.Node, bool)

	// DependenciesOf retrieves all nodes that the given node directly depends on.
	//
	// This is a convenience method that:
	//   1. Queries topology store for dependency node IDs
	//   2. Looks up each dependency node's full configuration
	//   3. Returns the complete node objects (not just IDs)
	//
	// Used by scheduler to analyze which dependencies must complete before a node can run.
	//
	// Returns:
	//   - Slice of dependency nodes (empty if node has no dependencies)
	//   - Error if the node doesn't exist or lookup fails
	//
	// Thread-safety: Must be safe to call concurrently.
	DependenciesOf(ctx context.Context, id nodeid.Address) ([]*node.Node, error)

	// NodeStatus retrieves the current execution status of a node.
	//
	// Returns the status and true if found, or StatusPending and false if the node
	// hasn't been initialized yet.
	//
	// Used by scheduler to determine which nodes have completed and which dependencies
	// are satisfied.
	//
	// Possible statuses: Pending, Running, Completed, Failed, Skipped
	//
	// Thread-safety: Must be safe to call concurrently.
	NodeStatus(ctx context.Context, id nodeid.Address) (node.Status, bool)

	// AllNodes returns all nodes registered in the topology.
	//
	// Used by scheduler to discover the full set of nodes that need to be executed.
	//
	// The order of nodes in the returned slice is not guaranteed unless specified
	// by the implementation.
	//
	// Thread-safety: Must be safe to call concurrently. The returned slice should
	// be safe for the caller to iterate.
	AllNodes(ctx context.Context) []*node.Node

	// MarkRunning transitions a node to Running status.
	//
	// Called by executor immediately before starting node execution to prevent
	// the scheduler from emitting the same node twice.
	//
	// State transition: Pending → Running
	//
	// Thread-safety: Must be safe to call concurrently for different nodes.
	MarkRunning(ctx context.Context, id nodeid.Address) error

	// MarkCompleted transitions a node to Completed status and records its output.
	//
	// Called by executor after successful node execution. The output is stored
	// in the node store and can later be retrieved by builder for expression resolution.
	//
	// State transition: Running → Completed
	//
	// Thread-safety: Must be safe to call concurrently for different nodes.
	MarkCompleted(ctx context.Context, id nodeid.Address, output any) error

	// MarkFailed transitions a node to Failed status and records the error.
	//
	// Called by executor when node execution fails. The error is stored for
	// debugging and error reporting.
	//
	// State transition: Running → Failed
	//
	// Thread-safety: Must be safe to call concurrently for different nodes.
	MarkFailed(ctx context.Context, id nodeid.Address, nodeErr error) error

	// MarkSkipped transitions a node to Skipped status.
	//
	// Called by executor when a node should be skipped (e.g., its dependencies failed,
	// or conditional execution determined it shouldn't run).
	//
	// State transition: Pending → Skipped
	//
	// Thread-safety: Must be safe to call concurrently for different nodes.
	MarkSkipped(ctx context.Context, id nodeid.Address) error
}
