package dag

import (
	"sync"
	"sync/atomic"

	"github.com/vk/burstgridgo/internal/schema"
)

// --- Public Structs ---

// Graph is the collection of all nodes that represent the execution plan.
type Graph struct {
	Nodes map[string]*Node
}

// Node is a single node in the execution graph.
type Node struct {
	ID             string // Unique ID, e.g., "step.http_request.my_step"
	Name           string // The instance name from HCL
	Type           NodeType
	StepConfig     *schema.Step
	ResourceConfig *schema.Resource
	Deps           map[string]*Node
	Dependents     map[string]*Node
	Error          error
	Output         any

	// Internal state management
	depCount        atomic.Int32
	descendantCount atomic.Int32
	state           atomic.Int32
	destroyOnce     sync.Once
	skipOnce        sync.Once
}

// --- Enums and Consts ---

// NodeType distinguishes between node types in the graph.
type NodeType int

const (
	StepNode NodeType = iota
	ResourceNode
)

// State represents the execution state of a node in the graph.
type State int32

const (
	Pending State = iota
	Running
	Done
	Failed
)

// --- Concurrency-Safe Methods ---

// DepCount returns the current number of unmet dependencies.
func (n *Node) DepCount() int32 {
	return n.depCount.Load()
}

// DecrementDepCount atomically decrements the dependency counter and returns the new value.
func (n *Node) DecrementDepCount() int32 {
	return n.depCount.Add(-1)
}

// DecrementDescendantCount atomically decrements the resource descendant counter and returns the new value.
func (n *Node) DecrementDescendantCount() int32 {
	return n.descendantCount.Add(-1)
}

// SetState atomically sets the node's execution state.
func (n *Node) SetState(s State) {
	n.state.Store(int32(s))
}

// GetState atomically retrieves the node's execution state.
func (n *Node) GetState() State {
	return State(n.state.Load())
}

// Destroy executes the given cleanup function exactly once.
func (n *Node) Destroy(f func()) {
	n.destroyOnce.Do(f)
}

// Skip marks a node as failed, ensures its WaitGroup counter is decremented
// exactly once, and returns true if it was the first time being skipped.
func (n *Node) Skip(err error, wg *sync.WaitGroup) bool {
	var wasSkipped bool
	n.skipOnce.Do(func() {
		n.SetState(Failed)
		n.Error = err
		wg.Done()
		wasSkipped = true
	})
	return wasSkipped
}

// SetInitialCounters sets the initial values for the node's atomic counters.
// This should only be called during graph construction.
func (n *Node) SetInitialCounters() {
	n.depCount.Store(int32(len(n.Deps)))
	if n.Type == ResourceNode {
		var directStepDependents int32 = 0
		for _, dependent := range n.Dependents {
			if dependent.Type == StepNode {
				directStepDependents++
			}
		}
		n.descendantCount.Store(directStepDependents)
	}
}
