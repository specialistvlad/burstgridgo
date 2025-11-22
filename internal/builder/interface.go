// Package builder provides the task building logic that transforms graph nodes
// into fully-resolved, executable tasks by evaluating expressions and resolving inputs.
//
// # Why Builder Exists
//
// The builder is the bridge between declarative HCL configuration and executable runtime values.
// It resolves all expressions, evaluates references to other node outputs, and produces a
// concrete Task with all inputs ready for the handler to execute.
//
// This separation provides several architectural benefits:
//   - **Expression Isolation:** HCL expression evaluation happens once, not during handler execution
//   - **Testability:** Task building can be tested independently of handler execution
//   - **Clarity:** Handlers receive plain data structures, not HCL expressions
//   - **Flexibility:** Builder can implement caching, validation, or type coercion
//
// # Responsibilities
//
// The builder is responsible for:
//   - **Expression Resolution:** Evaluating HCL expressions like `step.first.output.value`
//   - **Dependency Output Lookup:** Querying the graph/node store for completed node outputs
//   - **Variable Substitution:** Replacing locals and variables with their values
//   - **Type Conversion:** Ensuring inputs match the expected types for handlers
//   - **Error Handling:** Detecting missing dependencies or invalid expressions
//
// # How It Works
//
// For each node the executor wants to run:
//  1. **Receive:** Node configuration with potentially unresolved expressions in arguments
//  2. **Analyze:** Identify all expressions that reference other nodes (e.g., `step.first.output`)
//  3. **Resolve:** Query graph for dependency outputs and evaluate expressions using go-cty
//  4. **Build:** Create a Task with fully-resolved ResolvedInputs map
//  5. **Return:** Hand task to executor for dispatch to handler
//
// Example:
//
//	Node config:   { url: step.http_request.first.output.redirect_url }
//	Builder queries graph for: step.http_request.first.output
//	Gets:          { redirect_url: "https://example.com/new" }
//	Evaluates:     step.http_request.first.output.redirect_url → "https://example.com/new"
//	Returns Task:  { ResolvedInputs: { url: "https://example.com/new" } }
//
// # Relationship with Other Components
//
//   - **Graph:** Builder queries graph to retrieve dependency node outputs
//   - **Executor:** Calls Build() for each ready node before execution
//   - **Handler:** Receives the fully-resolved Task.ResolvedInputs map
//   - **bggoexpr:** Uses expression evaluation logic to resolve HCL expressions
//
// # Typical Implementation
//
// See DefaultBuilder for the reference implementation. A complete implementation
// would use internal/bggoexpr to evaluate expressions against a context containing
// all completed node outputs.
package builder

import (
	"context"

	"github.com/specialistvlad/burstgridgo/internal/graph"
	"github.com/specialistvlad/burstgridgo/internal/node"
	"github.com/specialistvlad/burstgridgo/internal/task"
)

// Builder transforms a graph node into a fully-resolved, executable task.
//
// The builder is responsible for:
//   - **Input Resolution:** Evaluating all HCL expressions in node arguments
//   - **Dependency Lookup:** Querying graph for outputs of completed dependencies
//   - **Task Creation:** Producing a Task with concrete, ready-to-use values
//
// # Usage Pattern
//
// The executor calls Build() for each ready node before execution:
//
//	task, err := builder.Build(ctx, readyNode, graph)
//	if err != nil {
//	    // Expression resolution failed
//	}
//	output, err := handler.Execute(task.ResolvedInputs)
//
// # Error Conditions
//
// Build() returns an error when:
//   - Required dependency outputs are missing (node didn't complete yet)
//   - Expression evaluation fails (syntax error, type mismatch)
//   - Referenced nodes don't exist in the graph
//   - Type conversion fails (e.g., expecting number, got string)
//
// # Thread-Safety
//
// Build() must be thread-safe for concurrent calls on different nodes.
// Implementations should query the thread-safe graph interface and avoid
// shared mutable state.
type Builder interface {
	// Build transforms a node into an executable task by resolving all input expressions.
	//
	// This method:
	//  1. Extracts the node's arguments/configuration (from node.Config)
	//  2. Identifies expressions that reference other nodes (e.g., `step.first.output`)
	//  3. Queries the graph to retrieve outputs of completed dependencies
	//  4. Evaluates expressions using go-cty to produce concrete values
	//  5. Returns a Task with ResolvedInputs map ready for handler execution
	//
	// Parameters:
	//   - ctx: Context for logging and cancellation
	//   - n: The node to build a task for (contains config and arguments)
	//   - g: Graph interface for querying dependency outputs
	//
	// Returns:
	//   - Task with fully-resolved inputs
	//   - Error if expression resolution fails or dependencies are missing
	//
	// # Current Implementation Note
	//
	// DefaultBuilder currently returns an empty Task with no input resolution (placeholder).
	// A complete implementation would use internal/bggoexpr to evaluate expressions.
	//
	// Thread-safety: Must be safe to call concurrently for different nodes.
	Build(ctx context.Context, n *node.Node, g graph.Graph) (*task.Task, error)
}
