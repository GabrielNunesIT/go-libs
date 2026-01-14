package logger

import (
	"context"
	"os"
)

type loggerKey struct{}

//nolint:containedctx // This struct implements context.Context by embedding it
type contextWithLogger struct {
	context.Context
}

// NewContextWithLogger returns a new context with the logger attached to it.
func NewContextWithLogger(ctx context.Context, ilogger ILogger) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if _, ok := ctx.Value(loggerKey{}).(ILogger); ok {
		// Do not store disabled logger.
		return &contextWithLogger{ctx}
	}

	return &contextWithLogger{context.WithValue(ctx, loggerKey{}, ilogger)}
}

//nolint:revive // Explicit type is useful for documentation
var ctxFallbackLogger ILogger = NewConsoleLogger(os.Stdout)

// SetCtxFallbackLogger sets the fallback logger to be used when no logger is found in the context.
func SetCtxFallbackLogger(logger ILogger) {
	ctxFallbackLogger = logger
}

// FromCtx returns the Logger associated with the ctx. If no logger
// is associated, CtxFallbackLogger is returned. Use SetCtxFallbackLogger to change it.
//
//nolint:ireturn // Returns interface to hide implementation details
func FromCtx(ctx context.Context) ILogger {
	if l, ok := ctx.Value(loggerKey{}).(ILogger); ok {
		return l
	}
	return ctxFallbackLogger
}
