package graph

import (
	"context"

	"github.com/specialistvlad/burstgridgo/internal/ctxlog"
	"github.com/specialistvlad/burstgridgo/internal/node"
	"github.com/specialistvlad/burstgridgo/internal/nodeid"
	"github.com/specialistvlad/burstgridgo/internal/nodestore"
	"github.com/specialistvlad/burstgridgo/internal/topologystore"
)

// Manager provides a high-level, thread-safe interface to the execution graph
// by composing and orchestrating lower-level storage backends.
type Manager struct{}

// New creates a new graph manager.
func New(ts topologystore.Store, ns nodestore.Store) Graph {
	return &Manager{}
}

func (m *Manager) Node(ctx context.Context, id nodeid.Address) (*node.Node, bool) {
	ctxlog.FromContext(ctx).Debug("graph.Manager.Node called (placeholder)", "id", id.String())
	return nil, false
}
func (m *Manager) DependenciesOf(ctx context.Context, id nodeid.Address) ([]*node.Node, error) {
	ctxlog.FromContext(ctx).Debug("graph.Manager.DependenciesOf called (placeholder)", "id", id.String())
	return nil, nil
}
func (m *Manager) NodeStatus(ctx context.Context, id nodeid.Address) (node.Status, bool) {
	ctxlog.FromContext(ctx).Debug("graph.Manager.NodeStatus called (placeholder)", "id", id.String())
	return node.StatusPending, true
}
func (m *Manager) AllNodes(ctx context.Context) []*node.Node {
	ctxlog.FromContext(ctx).Debug("graph.Manager.AllNodes called (placeholder)")
	return nil
}
func (m *Manager) MarkRunning(ctx context.Context, id nodeid.Address) error {
	ctxlog.FromContext(ctx).Debug("graph.Manager.MarkRunning called (placeholder)", "id", id.String())
	return nil
}
func (m *Manager) MarkCompleted(ctx context.Context, id nodeid.Address, output any) error {
	ctxlog.FromContext(ctx).Debug("graph.Manager.MarkCompleted called (placeholder)", "id", id.String())
	return nil
}
func (m *Manager) MarkFailed(ctx context.Context, id nodeid.Address, nodeErr error) error {
	ctxlog.FromContext(ctx).Debug("graph.Manager.MarkFailed called (placeholder)", "id", id.String(), "error", nodeErr)
	return nil
}
func (m *Manager) MarkSkipped(ctx context.Context, id nodeid.Address) error {
	ctxlog.FromContext(ctx).Debug("graph.Manager.MarkSkipped called (placeholder)", "id", id.String())
	return nil
}
