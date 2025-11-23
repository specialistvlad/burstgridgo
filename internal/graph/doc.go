// Package graph provides a unified facade for managing the execution graph,
// combining static topology (DAG structure) and dynamic state (execution status).
//
// # Why Graph Package Exists
//
// The graph package serves as a facade that simplifies interaction with the dual-store
// architecture (topology + node state). Instead of requiring components to coordinate
// between two separate stores, the Graph interface provides a single, cohesive API.
//
// This design provides several architectural benefits:
//   - **Unified API:** Executor and scheduler interact with one clean interface
//   - **Encapsulation:** Hides the dual-store implementation detail from consumers
//   - **Convenience:** Combines data from both stores (e.g., DependenciesOf returns full nodes)
//   - **Flexibility:** Future implementations can add caching, event hooks, or validation
//   - **Clarity:** Business logic doesn't know about storage implementation details
//
// # Architecture: The Facade Pattern
//
// The Graph is a thin facade over two specialized stores:
//
//	┌─────────────────────────────────────┐
//	│           Graph Facade              │
//	│  (Unified API for executor/         │
//	│   scheduler to query & update)      │
//	└──────────┬────────────┬─────────────┘
//	           │            │
//	           ▼            ▼
//	  ┌────────────┐  ┌────────────┐
//	  │  Topology  │  │ Node State │
//	  │   Store    │  │   Store    │
//	  │ (Structure)│  │  (Status)  │
//	  └────────────┘  └────────────┘
//
// **Topology Store** (topologystore.Store):
//   - Manages the immutable DAG structure (nodes and dependency edges)
//   - Write-once during graph construction, read-many during execution
//   - Queried by: AllNodes(), Node(), DependenciesOf()
//
// **Node Store** (nodestore.Store):
//   - Manages mutable execution state (status, outputs, errors)
//   - Continuously updated throughout execution
//   - Queried by: NodeStatus()
//   - Updated by: MarkRunning(), MarkCompleted(), MarkFailed(), MarkSkipped()
//
// # Lifecycle
//
//  1. **Creation:** Session factory creates graph with topology and node stores injected
//  2. **Population:** Builder adds nodes and dependencies to topology store (not via Graph interface)
//  3. **Execution:** Executor and scheduler use Graph interface to query and update
//  4. **Disposal:** Graph is discarded when session ends
//
// # Usage Patterns
//
// **Scheduler** queries graph to find ready nodes:
//
//	nodes := graph.AllNodes(ctx)
//	for _, node := range nodes {
//	    deps, _ := graph.DependenciesOf(ctx, node.ID)
//	    status, _ := graph.NodeStatus(ctx, node.ID)
//	    // Determine if node is ready to run
//	}
//
// **Executor** updates graph as nodes execute:
//
//	graph.MarkRunning(ctx, nodeID)
//	output, err := handler.Execute(task)
//	if err != nil {
//	    graph.MarkFailed(ctx, nodeID, err)
//	} else {
//	    graph.MarkCompleted(ctx, nodeID, output)
//	}
//
// **Builder** looks up node configuration:
//
//	node, ok := graph.Node(ctx, nodeID)
//	// Use node.RawConfig to build task
//
// # Thread-Safety
//
// All Graph methods are thread-safe. Thread-safety is guaranteed by delegating
// to the underlying thread-safe stores (topologystore and nodestore).
//
// # Key Types
//
// **Graph** (interface.go): The main interface for interacting with the execution graph.
// Provides methods for querying structure, checking status, and updating state.
//
// **Manager** (graph.go): The reference implementation that composes inmemorytopology
// and inmemorystore, delegating all operations to the appropriate store.
package graph
