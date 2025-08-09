package builder

import (
	"context"
	"sync"

	"github.com/vk/burstgridgo/internal/ctxlog"
)

// Dependencies returns a slice of Nodes that the given node directly depends on.
// It queries the underlying generic DAG and converts the returned string IDs into
// rich *Node pointers used by the application.
func (g *Graph) Dependencies(nodeID string) ([]*Node, error) {
	depIDs, err := g.dag.Dependencies(nodeID)
	if err != nil {
		return nil, err
	}
	deps := make([]*Node, 0, len(depIDs))
	for _, id := range depIDs {
		// This lookup is safe because the graph is static after being built.
		if node, ok := g.Nodes[id]; ok {
			deps = append(deps, node)
		}
	}
	return deps, nil
}

// Dependents returns a slice of Nodes that directly depend on the given node.
// It queries the underlying generic DAG and converts the returned string IDs into
// rich *Node pointers used by the application.
func (g *Graph) Dependents(nodeID string) ([]*Node, error) {
	depIDs, err := g.dag.Dependents(nodeID)
	if err != nil {
		return nil, err
	}
	deps := make([]*Node, 0, len(depIDs))
	for _, id := range depIDs {
		// This lookup is safe because the graph is static after being built.
		if node, ok := g.Nodes[id]; ok {
			deps = append(deps, node)
		}
	}
	return deps, nil
}

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

// SetInitialCounters prepares a node for the executor by setting its atomic
// counters based on the final graph topology.
func (n *Node) SetInitialCounters(ctx context.Context, g *Graph) error {
	logger := ctxlog.FromContext(ctx).With("node_id", n.ID)

	deps, err := g.Dependencies(n.ID)
	if err != nil {
		return err
	}
	depCount := int32(len(deps))
	n.depCount.Store(depCount)
	logger.Debug("Initialized dependency counter.", "count", depCount)

	if n.Type == ResourceNode {
		dependents, err := g.Dependents(n.ID)
		if err != nil {
			return err
		}
		var directStepDependents int32 = 0
		for _, dependent := range dependents {
			if dependent.Type == StepNode {
				directStepDependents++
			}
		}
		n.descendantCount.Store(directStepDependents)
		logger.Debug("Initialized resource descendant counter.", "count", directStepDependents)
	}
	return nil
}
