// Package node defines the core data structures for a single unit of work in the graph.
package node

import "github.com/specialistvlad/burstgridgo/internal/nodeid"

// Node represents a single, static definition of a unit of work in the DAG.
// It contains the information parsed from the HCL configuration but does not
// hold dynamic, run-time state such as its execution status or output.
type Node struct {
	// ID is the structured, unique identifier for this node within the graph.
	ID nodeid.Address

	// Type is the string that maps this node to a specific runner or handler
	// implementation in the registry (e.g., "http_request", "print").
	Type string

	// RawConfig holds the raw, unprocessed configuration for this node,
	// which will be decoded and validated by a specific handler.
	RawConfig map[string]any

	// Dependencies holds the string addresses of the nodes that must be
	// successfully completed before this node can run.
	Dependencies []string
}

// Status represents the execution state of a Node during a run.
type Status string

const (
	// StatusPending indicates the node has not yet been processed. This is the default state.
	StatusPending Status = "pending"

	// StatusRunning indicates the node is currently being executed by a runner.
	StatusRunning Status = "running"

	// StatusCompleted indicates the node finished execution successfully.
	StatusCompleted Status = "completed"

	// StatusFailed indicates the node terminated with an error.
	StatusFailed Status = "failed"

	// StatusSkipped indicates the node's execution was skipped, often due to
	// a failed dependency or a conditional check.
	StatusSkipped Status = "skipped"
)
