// Package scheduler provides the scheduling logic for determining which nodes
// in the execution DAG are ready to run based on dependency satisfaction.
//
// # Why Scheduler Exists
//
// The scheduler is the core engine that enables parallel execution in BurstGridGo.
// It continuously analyzes the dependency graph and node execution state to determine
// which nodes can run next, enabling maximum parallelization while respecting dependencies.
//
// This provides several key benefits:
//   - **Automatic Parallelization:** Executes independent nodes concurrently without explicit threading
//   - **Dependency Safety:** Ensures nodes only run after all dependencies complete successfully
//   - **Execution Efficiency:** Maximizes CPU utilization by finding all ready nodes at each step
//   - **Decoupled Logic:** Separates "what can run" (scheduler) from "how to run it" (executor)
//
// # How It Works
//
// The scheduler follows a continuous cycle:
//   1. Query graph for all nodes and their current status
//   2. Find nodes where all dependencies are Completed
//   3. Emit those nodes via ReadyNodes() channel
//   4. Wait for executor to mark nodes as Running/Completed/Failed
//   5. Repeat until no more nodes are ready (all Completed/Failed or waiting on dependencies)
//
// # Relationship with Other Components
//
//   - **Graph:** Scheduler queries the graph to check node status and dependencies
//   - **Executor:** Consumes ReadyNodes() channel and executes each node
//   - **Node Store:** Indirectly queried via graph to track execution state
//
// # Typical Implementation
//
// See DefaultScheduler for the reference implementation. A complete implementation
// would run a background goroutine that watches the graph and emits ready nodes.
package scheduler

import "github.com/specialistvlad/burstgridgo/internal/node"

// Scheduler analyzes the dependency graph and node execution state to determine
// which nodes are ready for execution.
//
// The scheduler is responsible for:
//   - **Dependency Analysis:** Checking which nodes have all dependencies satisfied
//   - **Ready Detection:** Finding nodes in Pending status whose dependencies are all Completed
//   - **Streaming:** Emitting ready nodes via a channel as they become available
//   - **Termination:** Closing the channel when the graph reaches a terminal state
//
// # Usage Pattern
//
// The executor consumes the ReadyNodes() channel in a loop:
//
//	for node := range scheduler.ReadyNodes() {
//	    // Execute node in a goroutine
//	    go executor.executeNode(node)
//	}
//
// # Terminal States
//
// The scheduler closes the ReadyNodes() channel when the graph reaches a terminal state:
//   - **Success:** All nodes are Completed
//   - **Partial Failure:** Some nodes Failed, remaining nodes can't run (dependencies unsatisfied)
//   - **Deadlock:** No nodes are ready, but some are still Pending (cyclic dependency or bug)
//
// # Thread-Safety
//
// The ReadyNodes() channel provides thread-safe communication between scheduler and executor.
// The scheduler internally queries the thread-safe graph, ensuring correct concurrent access.
type Scheduler interface {
	// ReadyNodes returns a read-only channel that streams nodes as they become ready for execution.
	//
	// A node is "ready" when:
	//   - Its status is Pending (not yet started)
	//   - All of its dependencies have status Completed (successfully finished)
	//
	// The scheduler runs in a background goroutine and continuously:
	//   1. Scans the graph for ready nodes
	//   2. Emits them via the returned channel
	//   3. Waits for state changes (nodes completing/failing)
	//   4. Repeats until terminal state
	//
	// The channel is **closed by the scheduler** when the graph reaches a terminal state
	// (all nodes completed/failed, or no more nodes can run).
	//
	// The executor MUST consume this channel promptly to avoid blocking the scheduler.
	//
	// # Current Implementation Note
	//
	// DefaultScheduler currently returns an immediately-closed channel (placeholder).
	// A complete implementation would run a goroutine that watches the graph and emits
	// nodes dynamically.
	ReadyNodes() <-chan *node.Node
}
