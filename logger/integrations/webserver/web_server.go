// Package webserver provides an Echo middleware for logging.
package webserver

import (
	"crypto/rand"
	"math/big"
	"net/http"
	"os"

	"github.com/GabrielNunesIT/go-libs/logger"
	webserver "github.com/GabrielNunesIT/go-libs/web-server"
	"github.com/labstack/echo/v4"
)

// NewLogger creates a new WebServer logger middleware.
func NewLogger(opts ...Option) *Logger {
	logInstance := &Logger{
		Logger: &webserver.Logger{
			ILogger: logger.NewConsoleLogger(os.Stdout),
		},
		config: loggerConfig{
			levelToUse: logger.LevelInfo,
		},
	}
	logInstance.SetPrefix("[ECHO]")

	for _, opt := range opts {
		opt(logInstance)
	}

	return logInstance
}

// WithJSONLogger configures the middleware to use structured JSON output
// instead of the default human-readable console format.
func WithJSONLogger() Option {
	return func(el *Logger) {
		currentPrefix := el.Prefix()
		el.Logger = &webserver.Logger{
			ILogger: logger.NewJSONLogger(os.Stdout),
		}
		el.SetPrefix(currentPrefix)
	}
}

func setRequestID(req *http.Request, res *echo.Response, config *loggerConfig) {
	if config.logRequestIDHeader == "" {
		config.logRequestIDHeader = "X-Request-ID"
	}
	rid := req.Header.Get(config.logRequestIDHeader)
	if rid == "" {
		rid = generateRandomRequestID(12)
	}
	res.Header().Set(config.logRequestIDHeader, rid)
}

func generateRandomRequestID(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	buf := make([]byte, length)
	for i := range buf {
		num, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		buf[i] = charset[num.Int64()]
	}

	return string(buf)
}
