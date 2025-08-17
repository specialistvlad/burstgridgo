package builder

import (
	"context"

	"github.com/specialistvlad/burstgridgo/internal/graph"
	"github.com/specialistvlad/burstgridgo/internal/node"
	"github.com/specialistvlad/burstgridgo/internal/task"
)

// Builder prepares a runnable Task from a graph node by resolving its inputs.
// This is called by the Executor just before a node is run.
type Builder interface {
	Build(ctx context.Context, n *node.Node, g graph.Graph) (*task.Task, error)
}
