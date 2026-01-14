package logger

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultLogger(t *testing.T) {
	// Save original default logger
	original := GetDefaultLogger()
	defer SetDefaultLogger(original)

	var buf bytes.Buffer
	l := NewConsoleLogger(&buf)
	l.SetLevel(LevelTrace)

	SetDefaultLogger(l)
	assert.Equal(t, l, GetDefaultLogger())

	Trace("trace msg")
	assert.Contains(t, buf.String(), "trace msg")

	Debug("debug msg")
	assert.Contains(t, buf.String(), "debug msg")

	Info("info msg")
	assert.Contains(t, buf.String(), "info msg")

	Warning("warn msg")
	assert.Contains(t, buf.String(), "warn msg")

	Error("error msg")
	assert.Contains(t, buf.String(), "error msg")

	assert.Panics(t, func() {
		Panic("panic msg")
	})
	assert.Contains(t, buf.String(), "panic msg")

	// Test SetDefaultLogger with nil
	SetDefaultLogger(nil)
	assert.Equal(t, l, GetDefaultLogger()) // Should not change
}
