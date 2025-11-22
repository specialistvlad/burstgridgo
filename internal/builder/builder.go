package builder

import (
	"context"

	"github.com/specialistvlad/burstgridgo/internal/ctxlog"
	"github.com/specialistvlad/burstgridgo/internal/graph"
	"github.com/specialistvlad/burstgridgo/internal/node"
	"github.com/specialistvlad/burstgridgo/internal/task"
)

// DefaultBuilder is the reference implementation of the Builder interface.
//
// # Current Status: Stubbed
//
// This implementation currently returns an empty Task with no input resolution (placeholder).
// A complete implementation would:
//
//  1. Extract node configuration and arguments from node.Config
//  2. Use internal/bggoexpr to identify expressions (e.g., `step.first.output.value`)
//  3. For each expression referencing another node:
//     a. Parse the node ID (e.g., "step.first")
//     b. Query graph.Node() or underlying nodestore for the output
//     c. Navigate the output structure to get the referenced field (e.g., "value")
//  4. Evaluate expressions using go-cty's evaluation context
//  5. Build a map[string]any with all resolved values
//  6. Return Task{Node: n, ResolvedInputs: resolvedMap}
//
// # Example Resolution
//
// Given node config:
//
//	arguments = {
//	  url      = "https://api.example.com"
//	  auth_token = step.get_token.first.output.token
//	  retry    = local.max_retries
//	}
//
// Builder would:
//  1. Identify `step.get_token.first.output.token` expression
//  2. Query graph for node "step.get_token.first"
//  3. Retrieve its output: {token: "abc123", expires: 3600}
//  4. Extract .token field: "abc123"
//  5. Resolve local.max_retries from evaluation context
//  6. Return ResolvedInputs:
//     {
//       url: "https://api.example.com",
//       auth_token: "abc123",
//       retry: 3
//     }
//
// # Thread-Safety
//
// Thread-safety is guaranteed by:
//   - No shared mutable state (stateless)
//   - Delegating to thread-safe graph interface for queries
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
