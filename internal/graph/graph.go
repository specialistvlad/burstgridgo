package graph

import (
	"context"

	"github.com/specialistvlad/burstgridgo/internal/ctxlog"
	"github.com/specialistvlad/burstgridgo/internal/node"
	"github.com/specialistvlad/burstgridgo/internal/nodeid"
	"github.com/specialistvlad/burstgridgo/internal/nodestore"
	"github.com/specialistvlad/burstgridgo/internal/topologystore"
)

// Manager is the reference implementation of the Graph interface.
//
// It acts as a facade that composes topologystore.Store and nodestore.Store,
// providing a unified API for the executor and scheduler to interact with both
// structure and state without knowing about the underlying dual-store architecture.
//
// The Manager delegates operations to the appropriate store:
//   - Structure queries (Node, AllNodes, DependenciesOf) → topology store
//   - State queries (NodeStatus) → node store
//   - State updates (MarkRunning, MarkCompleted, etc.) → node store
//
// # Thread-Safety
//
// Thread-safety is guaranteed by delegating to the underlying thread-safe stores
// (topologystore and nodestore). The Manager itself is stateless except for the
// store references.
type Manager struct {
	topology  topologystore.Store // Manages static DAG structure
	nodeState nodestore.Store     // Manages mutable execution state
}

// New creates a new graph manager that delegates to the provided stores.
func New(ts topologystore.Store, ns nodestore.Store) Graph {
	return &Manager{
		topology:  ts,
		nodeState: ns,
	}
}

// Node retrieves a node's configuration from the topology store.
func (m *Manager) Node(ctx context.Context, id nodeid.Address) (*node.Node, bool) {
	return m.topology.GetNode(ctx, id)
}

// DependenciesOf retrieves all nodes that the given node directly depends on.
// This method combines data from both stores: it queries the topology for dependency
// IDs, then looks up the full node configuration for each dependency.
func (m *Manager) DependenciesOf(ctx context.Context, id nodeid.Address) ([]*node.Node, error) {
	// Get dependency IDs from topology store
	depIDs, err := m.topology.DependenciesOf(ctx, id)
	if err != nil {
		return nil, err
	}

	// Look up full node configuration for each dependency
	deps := make([]*node.Node, 0, len(depIDs))
	for _, depID := range depIDs {
		depNode, ok := m.topology.GetNode(ctx, depID)
		if !ok {
			// This shouldn't happen if topology is consistent, but handle gracefully
			logger := ctxlog.FromContext(ctx)
			logger.Warn("dependency node not found in topology", "node", id.String(), "dependency", depID.String())
			continue
		}
		deps = append(deps, depNode)
	}

	return deps, nil
}

// NodeStatus retrieves the current execution status from the node store.
func (m *Manager) NodeStatus(ctx context.Context, id nodeid.Address) (node.Status, bool) {
	status, err := m.nodeState.GetStatus(ctx, id)
	if err != nil {
		// If there's an error, return Pending as the safe default
		logger := ctxlog.FromContext(ctx)
		logger.Warn("failed to get node status", "node", id.String(), "error", err)
		return node.StatusPending, false
	}
	return status, true
}

// AllNodes returns all nodes from the topology store.
func (m *Manager) AllNodes(ctx context.Context) []*node.Node {
	return m.topology.AllNodes(ctx)
}

// MarkRunning transitions a node to Running status in the node store.
func (m *Manager) MarkRunning(ctx context.Context, id nodeid.Address) error {
	return m.nodeState.SetStatus(ctx, id, node.StatusRunning)
}

// MarkCompleted transitions a node to Completed status and records its output.
func (m *Manager) MarkCompleted(ctx context.Context, id nodeid.Address, output any) error {
	if err := m.nodeState.SetStatus(ctx, id, node.StatusCompleted); err != nil {
		return err
	}
	return m.nodeState.SetOutput(ctx, id, output)
}

// MarkFailed transitions a node to Failed status and records the error.
func (m *Manager) MarkFailed(ctx context.Context, id nodeid.Address, nodeErr error) error {
	if err := m.nodeState.SetStatus(ctx, id, node.StatusFailed); err != nil {
		return err
	}
	return m.nodeState.SetError(ctx, id, nodeErr)
}

// MarkSkipped transitions a node to Skipped status.
func (m *Manager) MarkSkipped(ctx context.Context, id nodeid.Address) error {
	return m.nodeState.SetStatus(ctx, id, node.StatusSkipped)
}
