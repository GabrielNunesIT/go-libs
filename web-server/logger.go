package webserver

import (
	"fmt"
	"io"
	"os"

	"github.com/GabrielNunesIT/go-libs/logger"
	"github.com/labstack/gommon/log"
)

type (
	// Logger is an Echo logger implementation.
	Logger struct {
		logger.ILogger
		prefix string
	}
)

// Output returns the logger output.
func (e *Logger) Output() io.Writer {
	if len(e.GetOutput()) > 0 {
		return e.GetOutput()[0]
	}
	return nil
}

// SetOutput sets the logger output.
func (e *Logger) SetOutput(w io.Writer) {
	e.ILogger.SetOutput(w)
}

// Prefix returns the logger prefix.
func (e *Logger) Prefix() string {
	return e.prefix
}

// SetPrefix sets the logger prefix.
func (e *Logger) SetPrefix(p string) {
	e.prefix = p
}

// Level returns the logger level.
func (e *Logger) Level() log.Lvl {
	switch e.GetLevel() {
	case logger.LevelTrace, logger.LevelDebug:
		return log.DEBUG
	case logger.LevelInfo:
		return log.INFO
	case logger.LevelWarning:
		return log.WARN
	case logger.LevelError:
		return log.ERROR
	case logger.LevelPanic:
		return log.ERROR
	default:
		return log.ERROR
	}
}

// SetLevel sets the logger level.
func (e *Logger) SetLevel(l log.Lvl) {
	switch l {
	case log.DEBUG:
		e.ILogger.SetLevel(logger.LevelDebug)
	case log.INFO:
		e.ILogger.SetLevel(logger.LevelInfo)
	case log.WARN:
		e.ILogger.SetLevel(logger.LevelWarning)
	case log.ERROR:
		e.ILogger.SetLevel(logger.LevelError)
	case log.OFF:
		e.ILogger.SetLevel(logger.LevelInfo)
	default:
		e.ILogger.SetLevel(logger.LevelInfo)
	}
}

// SetHeader sets the logger header.
func (e *Logger) SetHeader(h string) {
	// do nothing
	_ = h
}

// Print prints a message.
func (e *Logger) Print(i ...interface{}) {
	e.Info(fmt.Sprint(i...))
}

// Printf prints a formatted message.
func (e *Logger) Printf(format string, args ...interface{}) {
	e.Info(fmt.Sprintf(format, args...))
}

// Printj prints a JSON message.
func (e *Logger) Printj(j log.JSON) {
	e.Info(fmt.Sprintf("%v", j))
}

// Debug prints a debug message.
func (e *Logger) Debug(i ...interface{}) {
	e.ILogger.Debug(fmt.Sprint(i...))
}

// Debugf prints a formatted debug message.
func (e *Logger) Debugf(format string, args ...interface{}) {
	e.ILogger.Debugf(format, args...)
}

// Debugj prints a JSON debug message.
func (e *Logger) Debugj(j log.JSON) {
	e.ILogger.Debug(fmt.Sprintf("%v", j))
}

// Info prints an info message.
func (e *Logger) Info(i ...interface{}) {
	e.ILogger.Info(fmt.Sprint(i...))
}

// Infof prints a formatted info message.
func (e *Logger) Infof(format string, args ...interface{}) {
	e.ILogger.Infof(format, args...)
}

// Infoj prints a JSON info message.
func (e *Logger) Infoj(j log.JSON) {
	e.ILogger.Info(fmt.Sprintf("%v", j))
}

// Warn prints a warning message.
func (e *Logger) Warn(i ...interface{}) {
	e.Warning(fmt.Sprint(i...))
}

// Warnf prints a formatted warning message.
func (e *Logger) Warnf(format string, args ...interface{}) {
	e.Warning(fmt.Sprintf(format, args...))
}

// Warnj prints a JSON warning message.
func (e *Logger) Warnj(j log.JSON) {
	e.Warning(fmt.Sprintf("%v", j))
}

// Error prints an error message.
func (e *Logger) Error(i ...interface{}) {
	e.ILogger.Error(fmt.Sprint(i...))
}

// Errorf prints a formatted error message.
func (e *Logger) Errorf(format string, args ...interface{}) {
	e.ILogger.Errorf(format, args...)
}

// Errorj prints a JSON error message.
func (e *Logger) Errorj(j log.JSON) {
	e.ILogger.Error(fmt.Sprintf("%v", j))
}

// Fatal prints a fatal message and exits.
func (e *Logger) Fatal(i ...interface{}) {
	e.ILogger.Error(fmt.Sprint(i...))
	os.Exit(1)
}

// Fatalf prints a formatted fatal message and exits.
func (e *Logger) Fatalf(format string, args ...interface{}) {
	e.ILogger.Errorf(format, args...)
	os.Exit(1)
}

// Fatalj prints a JSON fatal message and exits.
func (e *Logger) Fatalj(j log.JSON) {
	e.ILogger.Error(fmt.Sprintf("%v", j))
	os.Exit(1)
}

// Panic prints a panic message and panics.
func (e *Logger) Panic(i ...interface{}) {
	e.ILogger.Panic(fmt.Sprint(i...))
}

// Panicf prints a formatted panic message and panics.
func (e *Logger) Panicf(format string, args ...interface{}) {
	e.ILogger.Panicf(format, args...)
}

// Panicj prints a JSON panic message and panics.
func (e *Logger) Panicj(j log.JSON) {
	e.ILogger.Panic(fmt.Sprintf("%v", j))
}
