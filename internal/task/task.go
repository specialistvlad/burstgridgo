package task

import "github.com/specialistvlad/burstgridgo/internal/node"

// Task represents a node that is fully prepared for execution.
// It is the output of a builder.Builder and the input for a component
// that executes the business logic (e.g., a handler or runner).
type Task struct {
	// Node is the original node definition from the graph.
	Node *node.Node

	// ResolvedInputs contains the final, computed input values for the handler,
	// with all dependencies and references resolved.
	ResolvedInputs map[string]any
}
