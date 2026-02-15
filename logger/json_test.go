package logger

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewJSONLogger(t *testing.T) {
	var buf bytes.Buffer
	l := NewJSONLogger(&buf)
	assert.NotNil(t, l)
	assert.Equal(t, LevelInfo, l.GetLevel())
}

func TestJSONLogger_OutputFormat(t *testing.T) {
	var buf bytes.Buffer
	l := NewJSONLogger(&buf)
	l.Info("hello json")

	var parsed map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &parsed)
	require.NoError(t, err, "output must be valid JSON")

	assert.Equal(t, "hello json", parsed["message"])
	assert.Equal(t, "info", parsed["level"])
	assert.Contains(t, parsed, "time")
}

func TestJSONLogger_Levels(t *testing.T) {
	var buf bytes.Buffer
	l := NewJSONLogger(&buf)
	l.SetLevel(LevelTrace)

	tests := []struct {
		name     string
		fn       func(args ...any)
		fnf      func(format string, args ...any)
		expected string
	}{
		{"Trace", l.Trace, l.Tracef, "trace"},
		{"Debug", l.Debug, l.Debugf, "debug"},
		{"Info", l.Info, l.Infof, "info"},
		{"Warning", l.Warning, l.Warningf, "warn"},
		{"Error", l.Error, l.Errorf, "error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			tt.fn("msg")

			var parsed map[string]interface{}
			require.NoError(t, json.Unmarshal(buf.Bytes(), &parsed))
			assert.Equal(t, tt.expected, parsed["level"])

			buf.Reset()
			tt.fnf("formatted %s", "msg")

			require.NoError(t, json.Unmarshal(buf.Bytes(), &parsed))
			assert.Equal(t, tt.expected, parsed["level"])
			assert.Equal(t, "formatted msg", parsed["message"])
		})
	}
}

func TestJSONLogger_Fields(t *testing.T) {
	var buf bytes.Buffer
	l := NewJSONLogger(&buf)

	l.AddField("customKey", "customValue")
	l.SetLogID("req-001")
	l.Info("with fields")

	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &parsed))

	assert.Equal(t, "customValue", parsed["customKey"])
	assert.Equal(t, "req-001", parsed["LogID"])
}

func TestJSONLogger_NewLogger(t *testing.T) {
	var buf bytes.Buffer
	l := NewJSONLogger(&buf)

	sub := l.NewLogger("component:")
	sub.Info("test")

	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &parsed))
	assert.Equal(t, "[component:] test", parsed["message"])
}

func TestJSONLogger_GetOutput(t *testing.T) {
	var buf bytes.Buffer
	l := NewJSONLogger(&buf)

	outputs := l.GetOutput()
	require.Len(t, outputs, 1)
	assert.Equal(t, &buf, outputs[0])
}
