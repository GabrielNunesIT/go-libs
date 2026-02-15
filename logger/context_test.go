package logger

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewContextWithLogger(t *testing.T) {
	var buf bytes.Buffer
	l := NewConsoleLogger(&buf)
	ctx := context.Background()

	ctxWithLogger := NewContextWithLogger(ctx, l)
	assert.NotNil(t, ctxWithLogger)
}

func TestFromCtx(t *testing.T) {
	var buf bytes.Buffer
	l := NewConsoleLogger(&buf)
	ctx := context.Background()
	ctxWithLogger := NewContextWithLogger(ctx, l)

	lFromCtx := FromCtx(ctxWithLogger)
	assert.Equal(t, l, lFromCtx)
}

func TestFromCtx_Default(t *testing.T) {
	lDefault := FromCtx(context.Background())
	assert.NotNil(t, lDefault)
}

func TestSetCtxFallbackLogger(t *testing.T) {
	var bufFallback bytes.Buffer
	fallback := NewConsoleLogger(&bufFallback)
	SetCtxFallbackLogger(fallback)

	lFallback := FromCtx(context.Background())
	assert.Equal(t, fallback, lFallback)
}

func TestNewContextWithLogger_NilContext(t *testing.T) {
	var buf bytes.Buffer
	l := NewConsoleLogger(&buf)

	ctxNil := NewContextWithLogger(nil, l)
	assert.NotNil(t, ctxNil)
}

func TestNewContextWithLogger_PreservesExisting(t *testing.T) {
	var buf bytes.Buffer
	l := NewConsoleLogger(&buf)
	ctx := context.Background()
	ctxWithLogger := NewContextWithLogger(ctx, l)

	var bufFallback bytes.Buffer
	fallback := NewConsoleLogger(&bufFallback)

	ctxWithLogger2 := NewContextWithLogger(ctxWithLogger, fallback)
	assert.Equal(t, l, FromCtx(ctxWithLogger2))
}

func TestContext_LoggerMethod(t *testing.T) {
	var buf bytes.Buffer
	l := NewConsoleLogger(&buf)
	ctx := context.Background()
	ctxWithLogger := NewContextWithLogger(ctx, l)

	assert.Equal(t, l, FromCtx(ctxWithLogger))
}
