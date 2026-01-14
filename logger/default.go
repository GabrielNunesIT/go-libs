package logger

import (
	"os"
)

//nolint:revive // Explicit type is useful for documentation
var dLog ILogger = NewConsoleLogger(os.Stdout)

// SetDefaultLogger sets the default logger.
func SetDefaultLogger(logger ILogger) {
	if logger != nil {
		dLog = logger
	}
}

// GetDefaultLogger returns the default logger.
//
//nolint:ireturn // Returns interface to hide implementation details
func GetDefaultLogger() ILogger {
	return dLog
}

// Trace logs a message at the Trace level using the default logger.
func Trace(msg string, args ...interface{}) {
	dLog.Trace(msg, args)
}

// Debug logs a message at the Debug level using the default logger.
func Debug(msg string, args ...interface{}) {
	dLog.Debug(msg, args)
}

// Info logs a message at the Info level using the default logger.
func Info(msg string, args ...interface{}) {
	dLog.Info(msg, args)
}

// Warning logs a message at the Warning level using the default logger.
func Warning(msg string, args ...interface{}) {
	dLog.Warning(msg, args)
}

// Error logs a message at the Error level using the default logger.
func Error(msg string, args ...interface{}) {
	dLog.Error(msg, args)
}

// Panic logs a message at the Panic level using the default logger.
func Panic(msg string, args ...interface{}) {
	dLog.Panic(msg, args)
}
