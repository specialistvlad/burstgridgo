package builder

import (
	"sync"
	"sync/atomic"

	"github.com/vk/burstgridgo/internal/config"
	"github.com/vk/burstgridgo/internal/dag"
)

// Graph is the primary artifact of the builder. It represents the complete,
// validated execution plan as a collection of nodes and their dependencies.
type Graph struct {
	// Nodes provides a fast, ID-based lookup for any node in the graph.
	Nodes map[string]*Node

	// dag holds the generic graph topology and is used for all topological
	// operations like cycle detection and dependency querying. It is unexported
	// to ensure all interactions are mediated by the builder's methods.
	dag *dag.Graph
}

// Node is a single vertex in the execution graph, representing one unit of work
// (e.g., executing a step) or a stateful entity (e.g., a resource).
type Node struct {
	// ID is the unique, machine-readable identifier for the node.
	// Example: "step.http_request.my_api_call[0]"
	ID string
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
