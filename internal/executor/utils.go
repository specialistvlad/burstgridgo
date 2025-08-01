package executor

import (
	"fmt"
	"log/slog"
)

// formatValueForLogs is a helper to pretty-print values for logging.
func formatValueForLogs(v any) any {
	// In the future, this can be expanded. For now, it's a placeholder.
	if val, ok := v.(slog.LogValuer); ok {
		return val
	}
	return fmt.Sprintf("%+v", v)
}
