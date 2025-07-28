package dag

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/vk/burstgridgo/internal/ctxlog"
	"github.com/vk/burstgridgo/internal/registry"
	"github.com/vk/burstgridgo/internal/schema"
)

// --- Public Structs ---

// Executor runs the tasks in a graph concurrently.
type Executor struct {
	Graph             *Graph
	wg                sync.WaitGroup
	resourceInstances sync.Map
	cleanupStack      []func()
	cleanupMutex      sync.Mutex
	registry          *registry.Registry
	numWorkers        int
}

// Graph is the collection of all nodes that represent the execution plan.
type Graph struct {
	Nodes map[string]*Node
}

// --- Internal Structs ---

// Node is a single node in the execution graph.
type Node struct {
	ID              string // Unique ID, e.g., "step.http_request.my_step"
	Name            string // The instance name from HCL
	Type            NodeType
	StepConfig      *schema.Step
	ResourceConfig  *schema.Resource
	Deps            map[string]*Node
	Dependents      map[string]*Node
	depCount        atomic.Int32
	descendantCount atomic.Int32
	destroyOnce     sync.Once
	skipOnce        sync.Once // Correctly signal WaitGroup for skipped dependents
	State           atomic.Int32
	Error           error
	Output          any
}

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

// --- Constructors ---

// NewExecutor creates a new graph executor. It requires the handler registries
// to be provided explicitly, decoupling it from any global state.
func NewExecutor(
	graph *Graph,
	numWorkers int,
	reg *registry.Registry,
) *Executor {
	// Note: We use the default logger here as the executor-specific logger
	// will be extracted from the context in the Run method.
	if numWorkers <= 0 {
		numWorkers = 10 // Default to 10 if an invalid number is provided.
	}
	return &Executor{
		Graph:      graph,
		numWorkers: numWorkers,
		registry:   reg,
	}
}

// NewGraph builds a dependency graph from a user's grid configuration.
func NewGraph(ctx context.Context, config *schema.GridConfig) (*Graph, error) {
	// Use a background context for logging during the graph build phase.
	logger := ctxlog.FromContext(ctx)
	logger.Debug("NewGraph: Starting graph construction.")
	graph := &Graph{Nodes: make(map[string]*Node)}

	// First pass: create all nodes for steps and resources.
	createNodes(ctx, config, graph)
	logger.Debug("NewGraph: Node creation complete.", "node_count", len(graph.Nodes))

	// Second pass: link dependencies.
	if err := linkNodes(ctx, graph); err != nil {
		return nil, err
	}
	logger.Debug("NewGraph: Node linking complete.")

	// Third pass: initialize counters.
	initializeCounters(graph)
	logger.Debug("NewGraph: Counter initialization complete.")

	if err := graph.detectCycles(); err != nil {
		return nil, fmt.Errorf("error validating dependency graph: %w", err)
	}
	logger.Debug("NewGraph: Cycle detection passed.")

	logger.Debug("NewGraph: Graph construction successful.")
	return graph, nil
}
