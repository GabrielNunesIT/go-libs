package logger

import (
	"context"
	"os"
)

type loggerKey struct{}

// NewContextWithLogger returns a new context with the logger attached to it.
func NewContextWithLogger(ctx context.Context, ilogger ILogger) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if _, ok := ctx.Value(loggerKey{}).(ILogger); ok {
		// Do not store disabled logger.
		return ctx
	}

	return context.WithValue(ctx, loggerKey{}, ilogger)
}

// Explicit type is useful for documentation
var ctxFallbackLogger ILogger = NewConsoleLogger(os.Stdout)

// SetCtxFallbackLogger sets the fallback logger to be used when no logger is found in the context.
func SetCtxFallbackLogger(logger ILogger) {
	ctxFallbackLogger = logger
}

// FromCtx returns the Logger associated with the ctx. If no logger
// is associated, CtxFallbackLogger is returned. Use SetCtxFallbackLogger to change it.
func FromCtx(ctx context.Context) ILogger {
	if l, ok := ctx.Value(loggerKey{}).(ILogger); ok {
		return l
	}
	return ctxFallbackLogger
}
