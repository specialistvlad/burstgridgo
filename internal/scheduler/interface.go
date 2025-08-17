package scheduler

import "github.com/specialistvlad/burstgridgo/internal/node"

// Scheduler analyzes the dependency graph and the state of its nodes to
// determine the next set of nodes that are ready to be executed.
type Scheduler interface {
	// ReadyNodes returns a read-only channel that streams nodes as they
	// become ready for execution. The channel is closed by the Scheduler
	// once the graph has reached a terminal state.
	ReadyNodes() <-chan *node.Node
}
