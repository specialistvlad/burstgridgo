package executor

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/vk/burstgridgo/internal/ctxlog"
	"github.com/vk/burstgridgo/internal/dag"
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
func (e *Executor) skipDependents(ctx context.Context, node *dag.Node) {
	logger := ctxlog.FromContext(ctx)
	for _, dependent := range node.Dependents {
		err := fmt.Errorf("skipped due to upstream failure of '%s'", node.ID)
		wasSkipped := dependent.Skip(err, &e.wg)
		if wasSkipped {
			logger.Warn("Skipping dependent node due to upstream failure.", "nodeID", dependent.ID, "dependency", node.ID)
			e.skipDependents(ctx, dependent)
		}
	}
}
