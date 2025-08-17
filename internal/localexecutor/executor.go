// Package localexecutor provides a concrete, in-process implementation of the
// executor.Executor interface.
package localexecutor

import (
	"context"

	"github.com/specialistvlad/burstgridgo/internal/builder"
	"github.com/specialistvlad/burstgridgo/internal/ctxlog"
	"github.com/specialistvlad/burstgridgo/internal/executor"
	"github.com/specialistvlad/burstgridgo/internal/graph"
	"github.com/specialistvlad/burstgridgo/internal/handlers"
	"github.com/specialistvlad/burstgridgo/internal/scheduler"
)

// Executor implements the executor.Executor interface for local execution.
type Executor struct{}

// New creates a new local executor.
func New(
	sch scheduler.Scheduler,
	g graph.Graph,
	b builder.Builder,
	reg *handlers.Handlers,
) executor.Executor {
	return &Executor{}
}

func (e *Executor) Execute(ctx context.Context) error {
	logger := ctxlog.FromContext(ctx)
	logger.Debug("localexecutor.Executor.Execute called (placeholder)")
	return nil
}
