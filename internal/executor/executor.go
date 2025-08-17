// Package executor defines the interface for the DAG execution engine.
package executor

import "context"

// Executor is responsible for orchestrating the end-to-end execution of a DAG.
// It manages concurrency, interacts with the scheduler, and dispatches tasks.
type Executor interface {
	Execute(ctx context.Context) error
}
