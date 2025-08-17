// Package session defines the core interfaces for creating and managing an
// execution session. It abstracts away the details of local vs. remote execution.
package session

import (
	"context"

	"github.com/specialistvlad/burstgridgo/internal/executor"
	"github.com/specialistvlad/burstgridgo/internal/handlers"
	"github.com/specialistvlad/burstgridgo/internal/model"
)

// SessionFactory creates an execution Session. Different implementations can
// support various backends, such as local or distributed execution.
type SessionFactory interface {
	NewSession(
		ctx context.Context,
		cfg *model.Grid,
		reg *handlers.Handlers,
	) (Session, error)
}

// Session represents a single execution run and manages its lifecycle.
type Session interface {
	GetExecutor() (executor.Executor, error)
	// Close releases any resources held by the session. It accepts a context
	// to allow for graceful cleanup operations.
	Close(ctx context.Context) error
}
