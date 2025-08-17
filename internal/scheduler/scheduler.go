package scheduler

import (
	"context"

	"github.com/specialistvlad/burstgridgo/internal/ctxlog"
	"github.com/specialistvlad/burstgridgo/internal/graph"
	"github.com/specialistvlad/burstgridgo/internal/node"
)

// DefaultScheduler implements the scheduler.Scheduler interface.
type DefaultScheduler struct{}

// New creates a new default scheduler. It requires the graph it will be analyzing.
func New(g graph.Graph) Scheduler {
	return &DefaultScheduler{}
}

// ReadyNodes implements the Scheduler interface.
func (s *DefaultScheduler) ReadyNodes() <-chan *node.Node {
	// This method is special as it returns a channel and runs in the background.
	// Using context.Background() here is acceptable for a top-level goroutine
	// within the scheduler, but a real implementation would need a way to be cancelled.
	logger := ctxlog.FromContext(context.Background())
	logger.Debug("scheduler.ReadyNodes called (placeholder)")
	ch := make(chan *node.Node)
	close(ch)
	return ch
}
