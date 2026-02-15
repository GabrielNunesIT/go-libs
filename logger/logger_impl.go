package logger

import (
	"fmt"
	"io"

	"github.com/rs/zerolog"
)

type logger struct {
	logger zerolog.Logger

	hasLogID bool
	prefix   string
	outputs  []io.Writer
}

// Trace logs a message at the Trace level.
func (l *logger) Trace(args ...any) {
	l.logger.Trace().Msg(l.prefix + fmt.Sprint(args...))
}

// Tracef logs a formatted message at the Trace level.
func (l *logger) Tracef(format string, args ...any) {
	l.logger.Trace().Msgf(l.prefix+format, args...)
}

// Debug logs a message at the Debug level.
func (l *logger) Debug(args ...any) {
	l.logger.Debug().Msg(l.prefix + fmt.Sprint(args...))
}

// Debugf logs a formatted message at the Debug level.
func (l *logger) Debugf(format string, args ...any) {
	l.logger.Debug().Msgf(l.prefix+format, args...)
}

// Info logs a message at the Info level.
func (l *logger) Info(args ...any) {
	l.logger.Info().Msg(l.prefix + fmt.Sprint(args...))
}

// Infof logs a formatted message at the Info level.
func (l *logger) Infof(format string, args ...any) {
	l.logger.Info().Msgf(l.prefix+format, args...)
}

// Warning logs a message at the Warning level.
func (l *logger) Warning(args ...any) {
	l.logger.Warn().Msg(l.prefix + fmt.Sprint(args...))
}

// Warningf logs a formatted message at the Warning level.
func (l *logger) Warningf(format string, args ...any) {
	l.logger.Warn().Msgf(l.prefix+format, args...)
}

// Error logs a message at the Error level.
func (l *logger) Error(args ...any) {
	l.logger.Error().Msg(l.prefix + fmt.Sprint(args...))
}

// Errorf logs a formatted message at the Error level.
func (l *logger) Errorf(format string, args ...any) {
	l.logger.Error().Msgf(l.prefix+format, args...)
}

// Panic logs a message at the Panic level and panics.
func (l *logger) Panic(args ...any) {
	l.logger.Panic().Msg(l.prefix + fmt.Sprint(args...))
}

// Panicf logs a formatted message at the Panic level and panics.
func (l *logger) Panicf(format string, args ...any) {
	l.logger.Panic().Msgf(l.prefix+format, args...)
}

// SetLevel sets the logging level for the logger.
func (l *logger) SetLevel(level Level) {
	zerologLvl := zerolog.NoLevel
	switch level {
	case LevelTrace:
		zerologLvl = zerolog.TraceLevel
	case LevelDebug:
		zerologLvl = zerolog.DebugLevel
	case LevelInfo:
		zerologLvl = zerolog.InfoLevel
	case LevelWarning:
		zerologLvl = zerolog.WarnLevel
	case LevelError:
		zerologLvl = zerolog.ErrorLevel
	case LevelPanic:
		zerologLvl = zerolog.PanicLevel
	}
	l.logger = l.logger.Level(zerologLvl)
}

// GetLevel retrieves the current logging level of the logger.
func (l *logger) GetLevel() Level {
	switch l.logger.GetLevel().String() {
	case levelTraceStr:
		return LevelTrace
	case levelDebugStr:
		return LevelDebug
	case levelInfoStr:
		return LevelInfo
	case levelWarnStr:
		return LevelWarning
	case levelErrorStr:
		return LevelError
	case levelPanicStr:
		return LevelPanic
	default:
		return LevelInfo
	}
}

// SetOutput sets the output destinations for the logger.
func (l *logger) SetOutput(out ...io.Writer) {
	if len(out) == 1 {
		l.logger = l.logger.Output(out[0])
	} else {
		l.logger = l.logger.Output(zerolog.MultiLevelWriter(out...))
	}

	// Store outputs for later use
	l.outputs = []io.Writer{}
	l.outputs = append(l.outputs, out...)
}

// GetOutput retrieves the current output destinations of the logger.
func (l *logger) GetOutput() []io.Writer {
	return l.outputs
}

// AddField adds a custom field to the logger.
func (l *logger) AddField(key string, value interface{}) {
	l.logger = l.logger.With().Interface(key, value).Logger()
}

// SetLogID sets a unique identifier for the log entry if it hasn't been set already.
func (l *logger) SetLogID(value interface{}) {
	if !l.hasLogID {
		l.logger = l.logger.With().Interface("LogID", value).Logger()
	}
}

// NewLogger creates a new logger instance with a prefixed format.
//
//nolint:ireturn // Returning interface to match ILogger signature
func (l *logger) NewLogger(format string, args ...any) ILogger {
	newLogger := *l
	newLogger.prefix = fmt.Sprintf(l.prefix+"["+format+"] ", args...)

	return &newLogger
}
