// Package logger provides a logging interface and implementations.
package logger

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

// NewConsoleLogger returns a new Logger with a default configuration and provided out writer.
// This is a zerolog.ConsoleWriter with UTC time format.
//
//nolint:ireturn // Returns interface to hide implementation details
func NewConsoleLogger(out io.Writer) ILogger {
	writer := zerolog.ConsoleWriter{
		Out:              out,
		TimeFormat:       time.RFC3339,
		TimeLocation:     time.UTC,
		FormatLevel:      formatLevel,
		FormatTimestamp:  formatTimestamp,
		PartsOrder:       []string{"time", "level", "LogID", "loggerPkgInfo", "loggerFncInfo", "logPrefix", "message"},
		FieldsExclude:    []string{"loggerPkgInfo", "loggerFncInfo", "LogID"},
		FormatPrepare:    formatLogID,
		FormatFieldValue: removeNilFields,
	}

	zl := zerolog.New(writer).Level(zerolog.InfoLevel).With().Timestamp().Logger()
	return &logger{logger: zl}
}

func formatLogID(m map[string]interface{}) error {
	if ok := m["LogID"]; ok != nil {
		//nolint:forcetypeassert // LogID is guaranteed to be string by SetLogID
		m["LogID"] = "[" + m["LogID"].(string) + "]"
	}
	return nil
}

func formatTimestamp(input interface{}) string {
	return fmt.Sprintf("[%s]", input)
}

func removeNilFields(input interface{}) string {
	if input == nil {
		return ""
	}

	return fmt.Sprintf("%v", input)
}

func formatLevel(input interface{}) (s string) {
	s = "[%s]"
	if strLvl, ok := input.(string); ok {
		switch strLvl {
		case levelTraceStr:
			return fmt.Sprintf(s, "TRC")
		case levelDebugStr:
			return fmt.Sprintf(s, "DBG")
		case levelInfoStr:
			return fmt.Sprintf(s, "INF")
		case levelWarnStr:
			return fmt.Sprintf(s, "WRN")
		case levelErrorStr:
			return fmt.Sprintf(s, "ERR")
		case levelPanicStr:
			return fmt.Sprintf(s, "PNC")
		default:
			//nolint:forcetypeassert // input is guaranteed to be string
			return strings.ToUpper(fmt.Sprintf(s, input.(string)[:3]))
		}
	}

	return ""
}
