package old_executor

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/specialistvlad/burstgridgo/internal/ctxlog"
	"github.com/specialistvlad/burstgridgo/internal/node"
)

// formatValueForLogs is a helper to pretty-print values for logging.
func formatValueForLogs(v any) any {
	// In the future, this can be expanded. For now, it's a placeholder.
	if val, ok := v.(slog.LogValuer); ok {
		return val
	}
	return fmt.Sprintf("%+v", v)
}

// skipDependents recursively marks all downstream nodes as failed and decrements the WaitGroup.
func (e *Executor) skipDependents(ctx context.Context, node *node.Node) {
	logger := ctxlog.FromContext(ctx)

	// --- REFACTORED SECTION ---
	// We now query the graph for dependents instead of reading the legacy map.
	dependents, err := e.Graph.Dependents(node.ID())
	if err != nil {
		// This would be an unexpected internal error, as the node should always exist in the graph.
		logger.Error("Failed to get dependents while skipping nodes", "nodeID", node.ID(), "error", err)
		return
	}

	for _, dependent := range dependents {
		// --- END REFACTORED SECTION ---
		err := fmt.Errorf("skipped due to upstream failure of '%s'", node.ID())
		wasSkipped := dependent.Skip(err, &e.wg)
		if wasSkipped {
			logger.Warn("Skipping dependent node due to upstream failure.", "nodeID", dependent.ID(), "dependency", node.ID())
			e.skipDependents(ctx, dependent)
		}
	}
}
