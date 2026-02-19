package logger

import (
	"context"

	"github.com/rs/zerolog"
)

type ctxKey struct{}

var loggerKey = ctxKey{}

// Attach logger to context
func WithContext(ctx context.Context, log zerolog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, log)
}

// Get logger from context
func FromContext(ctx context.Context) zerolog.Logger {
	if log, ok := ctx.Value(loggerKey).(zerolog.Logger); ok {
		return log
	}

	// fallback logger if missing
	return zerolog.Nop()
}
