package webserver

import (
	"fmt"
	"time"

	"github.com/GabrielNunesIT/go-libs/logger"
	webserver "github.com/GabrielNunesIT/go-libs/web-server"
)

type (
	loggerConfig struct {
		// levelToUse instructs logger to use this log level for all log messages. If not set, logger will use `InfoLevel` as default.
		levelToUse logger.Level
		// logRequestID instructs logger to extract request ID from request `X-Request-ID` header or response if request did not have value.
		logRequestID bool
		// logRequestIDHeader is the header to extract the request ID from. If not set, logger will use `X-Request-ID` as default.
		logRequestIDHeader string
		// logProtocol instructs logger to extract request protocol (i.e. `HTTP/1.1` or `HTTP/2`)
		logProtocol bool
		// logMethod instructs logger to extract request method value (i.e. `GET` etc)
		logMethod bool
		// logURI instructs logger to extract request URI (i.e. `/list?lang=en&page=1`)
		logURI bool
		// logStatus instructs logger to extract response status code. If handler chain returns a web-server.HTTPError,
		// the status code is extracted from the web-server.HTTPError returned
		logStatus bool
		// logLatency instructs logger to record duration it took to execute rest of the handler chain (next(c) call).
		logLatency bool
	}

	// Logger is the WebServer logger middleware.
	Logger struct {
		*webserver.Logger
		config loggerConfig
	}

	// Option is a function that configures the Logger.
	Option func(*Logger)
)

// WithLogLevel sets the log level for the logger.
func WithLogLevel(level logger.Level) Option {
	return func(el *Logger) {
		el.config.levelToUse = level
	}
}

// WithLogRequestID sets the logger to log the request ID.
func WithLogRequestID() Option {
	return func(el *Logger) {
		el.config.logRequestID = true
	}
}

// WithLogRequestIDHeader sets the logger to log the request ID from the specified header.
func WithLogRequestIDHeader(header string) Option {
	return func(el *Logger) {
		el.config.logRequestID = true
		el.config.logRequestIDHeader = header
	}
}

// WithLogProtocol sets the logger to log the request protocol.
func WithLogProtocol() Option {
	return func(el *Logger) {
		el.config.logProtocol = true
	}
}

// WithLogMethod sets the logger to log the request method.
func WithLogMethod() Option {
	return func(el *Logger) {
		el.config.logMethod = true
	}
}

// WithLogURI sets the logger to log the request URI.
func WithLogURI() Option {
	return func(el *Logger) {
		el.config.logURI = true
	}
}

// WithLogStatus sets the logger to log the response status code.
func WithLogStatus() Option {
	return func(el *Logger) {
		el.config.logStatus = true
	}
}

// WithLogLatency sets the logger to log the request latency.
func WithLogLatency() Option {
	return func(el *Logger) {
		el.config.logLatency = true
	}
}

// WithLogger allows setting a custom logger instance.
func WithLogger(l *webserver.Logger) Option {
	return func(el *Logger) {
		el.Logger = l
	}
}

// ToMiddleware returns an Echo middleware that logs HTTP requests using the provided logger and configuration.
func (e *Logger) ToMiddleware() webserver.MiddlewareFunc {
	if e.config.levelToUse == 0 {
		e.config.levelToUse = logger.LevelInfo
	}

	return func(next webserver.HandlerFunc) webserver.HandlerFunc {
		return func(ctx webserver.Context) error {
			req := ctx.Request()
			res := ctx.Response()
			start := time.Now()

			// Apply request ID if needed
			if e.config.logRequestID {
				setRequestID(req, res, &e.config)

				id := req.Header.Get(e.config.logRequestIDHeader)
				if id == "" {
					id = res.Header().Get(e.config.logRequestIDHeader)
				}
				e.SetLogID(id)
			}

			// Add logger to context
			reqCtx := req.Context()
			ctx.SetRequest(req.WithContext(logger.NewContextWithLogger(reqCtx, e.ILogger)))

			msg := "New request:"
			if e.config.logProtocol {
				msg += " Protocol=" + req.Proto
			}
			if e.config.logMethod {
				msg += " Method=" + req.Method
			}
			if e.config.logURI {
				msg += " URI=" + req.RequestURI
			}

			if e.config.logStatus {
				statusCode := res.Status
				msg += fmt.Sprintf(" Status=%d", statusCode)
			}
			if e.config.logLatency {
				msg += fmt.Sprintf(" Latency=%d ms", time.Since(start).Milliseconds())
			}

			switch e.config.levelToUse {
			case logger.LevelTrace:
				e.Trace(msg)
			case logger.LevelDebug:
				e.Debug(msg)
			case logger.LevelInfo:
				e.Info(msg)
			case logger.LevelWarning, logger.LevelError, logger.LevelPanic:
				// do nothing
			}

			return next(ctx)
		}
	}
}
