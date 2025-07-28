package app

import (
	"io"
	"log/slog"
)

// newLogger creates and configures a new slog.Logger instance. It does not
// set the global logger, allowing for isolated logger instances.
func newLogger(levelStr, formatStr string, outW io.Writer) *slog.Logger {
	var level slog.Level
	switch levelStr {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	handlerOpts := &slog.HandlerOptions{Level: level}
	var handler slog.Handler

	if formatStr == "json" {
		handler = slog.NewJSONHandler(outW, handlerOpts)
	} else {
		handler = slog.NewTextHandler(outW, handlerOpts)
	}

	return slog.New(handler)
}
