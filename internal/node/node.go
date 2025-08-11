package node

import (
	"sync"
	"sync/atomic"

	"github.com/specialistvlad/burstgridgo/internal/config"
	"github.com/specialistvlad/burstgridgo/internal/nodeid"
)

// Node is a single vertex in the execution graph, representing one unit of work
// (e.g., executing a step) or a stateful entity (e.g., a resource).
type Node struct {
	// id is the unique, machine-readable, structured identifier for the node.
	id *nodeid.Address
	// Name is the human-readable instance name from the configuration.
	// Example: "my_api_call"
	Name string
	// Type distinguishes between nodes that represent steps and resources.
	Type NodeType

	// IsPlaceholder is true if this node represents a dynamic `count` or `for_each`
	// block. Such nodes are expanded at runtime by the executor.
	IsPlaceholder bool

	// StepConfig holds the configuration for a step node. It is nil for resources.
	StepConfig *config.Step
	// ResourceConfig holds the configuration for a resource node. It is nil for steps.
	ResourceConfig *config.Resource

	// Error stores any error that occurred during the node's execution.
	Error error
	// Output stores the result of the node's execution for use by downstream nodes.
	Output any

	// --- Internal state management ---

	// depCount is an atomic counter for unmet dependencies, used by the scheduler.
	depCount atomic.Int32
	// descendantCount is an atomic counter for a resource's step dependents,
	// used for efficient resource cleanup.
	descendantCount atomic.Int32
	// state is the node's current execution state, managed atomically.
	state atomic.Int32
	// destroyOnce ensures a node's cleanup/destruction logic is run exactly once.
	destroyOnce sync.Once
	// skipOnce ensures a node is marked as skipped and processed exactly once.
	skipOnce sync.Once
}

// ID returns the canonical string representation of the node's address.
func (n *Node) ID() string {
	return n.id.String()
}

// Address returns the structured address of the node.
func (n *Node) Address() *nodeid.Address {
	return n.id
}

func (n *Node) SetDepCount(count int32) {
	n.depCount.Store(count)
}

func (n *Node) SetDescCount(count int32) {
	n.descendantCount.Store(count)
}

// NodeType distinguishes between different kinds of nodes in the graph.
type NodeType int

const (
	// StepNode represents a node that executes a task.
	StepNode NodeType = iota
	// ResourceNode represents a node that manages a stateful resource.
	ResourceNode
)

// State represents the execution state of a node in the graph.
type State int32

const (
	// Pending indicates the node is waiting for its dependencies to complete.
	Pending State = iota
	// Running indicates the node is currently being executed by a worker.
	Running
	// Done indicates the node has completed execution successfully.
	Done
	// Failed indicates the node has failed execution or was skipped.
	Failed
)

// DepCount atomically returns the current number of unmet dependencies.
func (n *Node) DepCount() int32 {
	return n.depCount.Load()
}

// DecrementDepCount atomically decrements the dependency counter and returns the new value.
func (n *Node) DecrementDepCount() int32 {
	return n.depCount.Add(-1)
}

// DecrementDescendantCount atomically decrements the resource descendant counter.
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

// Destroy executes the given cleanup function exactly once, making it safe to
// call multiple times.
func (n *Node) Destroy(f func()) {
	n.destroyOnce.Do(f)
}

// Skip marks a node as failed and decrements its WaitGroup counter. It uses a
// sync.Once to guarantee this happens only once, returning true if it was the
// first time this node was skipped.
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

func CreateStepNode(id *nodeid.Address, config *config.Step, placeholder bool) *Node {
	Node := &Node{
		id:            id,
		Name:          config.Name,
		Type:          StepNode,
		IsPlaceholder: placeholder,
		StepConfig:    config,
	}
	return Node
}

func CreateResourceNode(id *nodeid.Address, config *config.Resource) *Node {
	Node := &Node{
		id:             id,
		Name:           config.Name,
		Type:           ResourceNode,
		ResourceConfig: config,
	}
	return Node
}
