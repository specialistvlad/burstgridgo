package builder

import (
	"context"

	"github.com/specialistvlad/burstgridgo/internal/ctxlog"
	"github.com/specialistvlad/burstgridgo/internal/graph"
	"github.com/specialistvlad/burstgridgo/internal/node"
	"github.com/specialistvlad/burstgridgo/internal/task"
)

// DefaultBuilder implements the logic for preparing a single node for execution.
type DefaultBuilder struct{}

// New creates a new default builder.
func New() Builder {
	return &DefaultBuilder{}
}

// Build implements the Builder interface.
func (b *DefaultBuilder) Build(ctx context.Context, n *node.Node, g graph.Graph) (*task.Task, error) {
	logger := ctxlog.FromContext(ctx)
	logger.Debug("builder.Build called (placeholder)", "node", n.ID.String())
	return &task.Task{Node: n, ResolvedInputs: make(map[string]any)}, nil
}
