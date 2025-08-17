// Package localsession provides a concrete implementation of the session.Session
// and session.SessionFactory interfaces for local, in-process execution.
package localsession

import (
	"context"

	"github.com/specialistvlad/burstgridgo/internal/builder"
	"github.com/specialistvlad/burstgridgo/internal/ctxlog"
	"github.com/specialistvlad/burstgridgo/internal/executor"
	"github.com/specialistvlad/burstgridgo/internal/graph"
	"github.com/specialistvlad/burstgridgo/internal/handlers"
	"github.com/specialistvlad/burstgridgo/internal/inmemorystore"
	"github.com/specialistvlad/burstgridgo/internal/inmemorytopology"
	"github.com/specialistvlad/burstgridgo/internal/localexecutor"
	"github.com/specialistvlad/burstgridgo/internal/model"
	"github.com/specialistvlad/burstgridgo/internal/scheduler"
	"github.com/specialistvlad/burstgridgo/internal/session"
)

// SessionFactory implements session.SessionFactory for local runs.
type SessionFactory struct{}

// NewSession creates and configures a new local session.
func (f *SessionFactory) NewSession(
	ctx context.Context,
	cfg *model.Grid,
	reg *handlers.Handlers,
) (session.Session, error) {
	logger := ctxlog.FromContext(ctx)
	logger.Debug("localsession.SessionFactory.NewSession called")

	// --- This is where the dependency injection wiring happens ---
	// For now, we are creating placeholder instances of each component.
	topoStore := inmemorytopology.New()
	nodeStore := inmemorystore.New()
	graph := graph.New(topoStore, nodeStore)
	taskBuilder := builder.New()
	sched := scheduler.New(graph)
	exec := localexecutor.New(sched, graph, taskBuilder, reg)
	// --- End of dependency injection ---

	return &Session{
		executor: exec,
	}, nil
}

// Session implements session.Session for local runs.
type Session struct {
	executor executor.Executor
}

// GetExecutor returns the executor that was created and wired up by the factory.
func (s *Session) GetExecutor() (executor.Executor, error) {
	// No context is needed here as this is a simple accessor.
	// Logging can be done by the caller if needed.
	return s.executor, nil
}

// Close uses the provided context for logging during cleanup.
func (s *Session) Close(ctx context.Context) error {
	logger := ctxlog.FromContext(ctx)
	logger.Debug("localsession.Session.Close called (placeholder)")
	return nil
}
