// Package ctxlog provides a context key for safely passing a slog.Logger
// instance through context.Context.
package ctxlog

import (
	"context"
	"log/slog"
)

// key is an unexported type to prevent collisions with context keys from other packages.
type key struct{}

// loggerKey is the key for the slog.Logger in a context.Context.
var loggerKey = key{}

// WithLogger returns a new context with the provided logger embedded.
func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

// FromContext extracts the slog.Logger from a context. If no logger is
// found, it returns the default global logger.
func FromContext(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(loggerKey).(*slog.Logger); ok {
		return logger
	}
	panic("ctxlog: logger missing from context")
}
