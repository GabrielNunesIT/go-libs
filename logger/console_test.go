package logger

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewConsoleLogger(t *testing.T) {
	var buf bytes.Buffer
	l := NewConsoleLogger(&buf)
	assert.NotNil(t, l)
	assert.Equal(t, LevelInfo, l.GetLevel())
}

func TestFormatLevel(t *testing.T) {
	assert.Equal(t, "[TRC]", formatLevel("trace"))
	assert.Equal(t, "[DBG]", formatLevel("debug"))
	assert.Equal(t, "[INF]", formatLevel("info"))
	assert.Equal(t, "[WRN]", formatLevel("warn"))
	assert.Equal(t, "[ERR]", formatLevel("error"))
	assert.Equal(t, "[PNC]", formatLevel("panic"))
	assert.Equal(t, "[UNK]", formatLevel("unknown")) // Should take first 3 chars and upper case
	assert.Equal(t, "", formatLevel(123))            // Not a string
}

func TestRemoveNilFields(t *testing.T) {
	assert.Equal(t, "", removeNilFields(nil))
	assert.Equal(t, "value", removeNilFields("value"))
}

func TestFormatTimestamp(t *testing.T) {
	assert.Equal(t, "[time]", formatTimestamp("time"))
}

func TestFormatLogID(t *testing.T) {
	m := make(map[string]interface{})
	m["LogID"] = "123"
	formatLogID(m)
	assert.Equal(t, "[123]", m["LogID"])

	m2 := make(map[string]interface{})
	formatLogID(m2)
	assert.Nil(t, m2["LogID"])
}
