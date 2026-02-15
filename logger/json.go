// Package logger provides a logging interface and implementations.
package logger

import (
	"io"

	"github.com/rs/zerolog"
)

// NewJSONLogger returns a new Logger that writes structured JSON output to the provided writer.
// The default level is InfoLevel, matching the Console logger behavior.
//
//nolint:ireturn // Returns interface to hide implementation details
func NewJSONLogger(out io.Writer) ILogger {
	zl := zerolog.New(out).Level(zerolog.InfoLevel).With().Timestamp().Logger()
	return &logger{
		logger:  zl,
		outputs: []io.Writer{out},
	}
}
