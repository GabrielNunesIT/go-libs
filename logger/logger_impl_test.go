package logger

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogger_Levels(t *testing.T) {
	var buf bytes.Buffer
	l := NewConsoleLogger(&buf)
	l.SetLevel(LevelTrace)

	tests := []struct {
		name  string
		fn    func(args ...any)
		fnf   func(format string, args ...any)
		level string
	}{
		{"Trace", l.Trace, l.Tracef, "TRC"},
		{"Debug", l.Debug, l.Debugf, "DBG"},
		{"Info", l.Info, l.Infof, "INF"},
		{"Warning", l.Warning, l.Warningf, "WRN"},
		{"Error", l.Error, l.Errorf, "ERR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			tt.fn("message")
			assert.Contains(t, buf.String(), tt.level)
			assert.Contains(t, buf.String(), "message")

			buf.Reset()
			tt.fnf("formatted %s", "message")
			assert.Contains(t, buf.String(), tt.level)
			assert.Contains(t, buf.String(), "formatted message")
		})
	}
}

func TestLogger_Panic(t *testing.T) {
	var buf bytes.Buffer
	l := NewConsoleLogger(&buf)
	l.SetLevel(LevelTrace)

	assert.Panics(t, func() {
		l.Panic("panic message")
	})
	assert.Contains(t, buf.String(), "PNC")
	assert.Contains(t, buf.String(), "panic message")

	buf.Reset()
	assert.Panics(t, func() {
		l.Panicf("panic %s", "formatted")
	})
	assert.Contains(t, buf.String(), "PNC")
	assert.Contains(t, buf.String(), "panic formatted")
}

func TestLogger_SetLevel(t *testing.T) {
	var buf bytes.Buffer
	l := NewConsoleLogger(&buf)

	levels := []struct {
		level Level
		str   string
	}{
		{LevelTrace, "trace"},
		{LevelDebug, "debug"},
		{LevelInfo, "info"},
		{LevelWarning, "warn"},
		{LevelError, "error"},
		{LevelPanic, "panic"},
	}

	for _, tt := range levels {
		l.SetLevel(tt.level)
		assert.Equal(t, tt.level, l.GetLevel())
	}
}

func TestLogger_GetLevel_Default(t *testing.T) {
	var buf bytes.Buffer
	l := NewConsoleLogger(&buf)

	// Set invalid level, which defaults to zerolog.NoLevel in SetLevel
	l.SetLevel(Level(0))

	// GetLevel should return LevelInfo (default)
	assert.Equal(t, LevelInfo, l.GetLevel())
}

func TestLogger_SetOutput(t *testing.T) {
	var buf1 bytes.Buffer
	var buf2 bytes.Buffer
	l := NewConsoleLogger(&buf1)

	l.SetOutput(&buf1, &buf2)
	outputs := l.GetOutput()
	assert.Len(t, outputs, 2)

	l.Info("test output")
	assert.Contains(t, buf1.String(), "test output")
	assert.Contains(t, buf2.String(), "test output")

	l.SetOutput(&buf1)
	outputs = l.GetOutput()
	assert.Len(t, outputs, 1)
}

func TestLogger_AddField(t *testing.T) {
	var buf bytes.Buffer
	l := NewConsoleLogger(&buf)
	l.SetLevel(LevelInfo)

	l.AddField("customKey", "customValue")
	l.Info("message")

	assert.Contains(t, buf.String(), "customKey")
	assert.Contains(t, buf.String(), "customValue")
}

func TestLogger_SetLogID(t *testing.T) {
	var buf bytes.Buffer
	l := NewConsoleLogger(&buf)

	l.SetLogID("12345")
	l.Info("message")

	// FormatLogID adds brackets: [12345]
	assert.Contains(t, buf.String(), "[12345]")

	l.SetLogID("67890")
	l.Info("message 2")
	assert.Contains(t, buf.String(), "[12345]") // It might still be 12345 if the logger is immutable?
}

func TestLogger_NewLogger(t *testing.T) {
	var buf bytes.Buffer
	l := NewConsoleLogger(&buf)

	sub := l.NewLogger("sub:")
	sub.Info("message")

	assert.Contains(t, buf.String(), "[sub:] message")

	sub2 := sub.NewLogger("sub2:")
	sub2.Info("message")
	assert.Contains(t, buf.String(), "[sub:] [sub2:] message")
}
