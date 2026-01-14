package logger

import (
	"io"
)

// Level represents the logging level.
type Level uint

// Logging levels.
const (
	LevelTrace Level = iota + 1
	LevelDebug
	LevelInfo
	LevelWarning
	LevelError
	LevelPanic
)

const (
	levelTraceStr = "trace"
	levelDebugStr = "debug"
	levelInfoStr  = "info"
	levelWarnStr  = "warn"
	levelErrorStr = "error"
	levelPanicStr = "panic"
)

// ILogger is the interface for the logger.
type ILogger interface {
	Trace(args ...any)
	Tracef(format string, args ...any)
	Debug(args ...any)
	Debugf(format string, args ...any)
	Info(args ...any)
	Infof(format string, args ...any)
	Warning(args ...any)
	Warningf(format string, args ...any)
	Error(args ...any)
	Errorf(format string, args ...any)
	Panic(args ...any)
	Panicf(format string, args ...any)

	SetLevel(level Level)
	GetLevel() Level

	SetOutput(out ...io.Writer)
	GetOutput() []io.Writer

	AddField(key string, value any)
	SetLogID(value any)

	SubLogger(format string, args ...any) ILogger
}
