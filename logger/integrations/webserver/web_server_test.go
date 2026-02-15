package webserver_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"testing"

	"github.com/GabrielNunesIT/go-libs/logger"
	wslogger "github.com/GabrielNunesIT/go-libs/logger/integrations/webserver"
	"github.com/GabrielNunesIT/go-libs/webserver"
	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
	"github.com/stretchr/testify/assert"
)

// MockLogger implements logger.ILogger for testing
type MockLogger struct {
	level  logger.Level
	output []io.Writer
}

func (m *MockLogger) Trace(args ...any)                                   {}
func (m *MockLogger) Tracef(format string, args ...any)                   {}
func (m *MockLogger) Debug(args ...any)                                   {}
func (m *MockLogger) Debugf(format string, args ...any)                   {}
func (m *MockLogger) Info(args ...any)                                    {}
func (m *MockLogger) Infof(format string, args ...any)                    {}
func (m *MockLogger) Warning(args ...any)                                 {}
func (m *MockLogger) Warningf(format string, args ...any)                 {}
func (m *MockLogger) Error(args ...any)                                   {}
func (m *MockLogger) Errorf(format string, args ...any)                   {}
func (m *MockLogger) Panic(args ...any)                                   {}
func (m *MockLogger) Panicf(format string, args ...any)                   {}
func (m *MockLogger) SetLevel(level logger.Level)                         { m.level = level }
func (m *MockLogger) GetLevel() logger.Level                              { return m.level }
func (m *MockLogger) SetOutput(out ...io.Writer)                          { m.output = out }
func (m *MockLogger) GetOutput() []io.Writer                              { return m.output }
func (m *MockLogger) AddField(key string, value any)                      {}
func (m *MockLogger) SetLogID(value any)                                  {}
func (m *MockLogger) NewLogger(format string, args ...any) logger.ILogger { return m }

func TestNewLogger(t *testing.T) {
	l := wslogger.NewLogger()
	assert.NotNil(t, l)
	assert.Equal(t, "[ECHO]", l.Prefix())
}

func TestOptions(t *testing.T) {
	l := wslogger.NewLogger(
		wslogger.WithLogLevel(logger.LevelDebug),
		wslogger.WithLogRequestID(),
		wslogger.WithLogRequestIDHeader("X-Custom-ID"),
		wslogger.WithLogProtocol(),
		wslogger.WithLogMethod(),
		wslogger.WithLogURI(),
		wslogger.WithLogStatus(),
		wslogger.WithLogLatency(),
	)
	assert.NotNil(t, l)
}

func TestMiddleware(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	l := wslogger.NewLogger(
		wslogger.WithLogRequestID(),
		wslogger.WithLogProtocol(),
		wslogger.WithLogMethod(),
		wslogger.WithLogURI(),
		wslogger.WithLogStatus(),
		wslogger.WithLogLatency(),
	)

	// Capture output
	var buf bytes.Buffer
	l.SetOutput(&buf)

	h := l.ToMiddleware()(func(c webserver.Context) error {
		return c.String(http.StatusOK, "test")
	})

	err := h(c)
	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "New request:")
	assert.Contains(t, buf.String(), "Protocol=")
	assert.Contains(t, buf.String(), "Method=GET")
	assert.Contains(t, buf.String(), "URI=/")
	// Status might be 0 if not properly set in the mock context/response interaction
	// But let's check if we can force it or if we should just check for "Status="
	assert.Contains(t, buf.String(), "Status=")
	assert.Contains(t, buf.String(), "Latency=")
}

func TestMiddleware_WithRequestID(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", "existing-id")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	l := wslogger.NewLogger(wslogger.WithLogRequestID())
	var buf bytes.Buffer
	l.SetOutput(&buf)

	h := l.ToMiddleware()(func(c webserver.Context) error {
		return c.String(http.StatusOK, "test")
	})

	err := h(c)
	assert.NoError(t, err)
	// Should use existing ID
	// We can't easily check the ID used in logger unless we check the log output for it?
	// But the logger doesn't log the ID in the message by default, it sets it in the logger context.
	// The ConsoleLogger might not print it unless configured?
	// But we just want to cover the code path.
}

func TestMiddleware_DefaultLevel(t *testing.T) {
	// Explicitly set level to 0 to trigger the default logic in ToMiddleware
	l := wslogger.NewLogger(wslogger.WithLogLevel(0))
	h := l.ToMiddleware()(func(c webserver.Context) error {
		return c.String(http.StatusOK, "test")
	})

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h(c)
	assert.NoError(t, err)
}

func TestMiddleware_LogLevels(t *testing.T) {
	tests := []struct {
		level logger.Level
		name  string
	}{
		{logger.LevelTrace, "Trace"},
		{logger.LevelDebug, "Debug"},
		{logger.LevelInfo, "Info"},
		{logger.LevelWarning, "Warning"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			l := wslogger.NewLogger(wslogger.WithLogLevel(tt.level))
			var buf bytes.Buffer
			l.SetOutput(&buf)
			// Set underlying logger level to Debug so everything is captured
			l.SetLevel(log.DEBUG)

			h := l.ToMiddleware()(func(c webserver.Context) error {
				return c.String(http.StatusOK, "test")
			})

			err := h(c)
			assert.NoError(t, err)

			if tt.level == logger.LevelWarning {
				assert.NotContains(t, buf.String(), "New request:")
			} else if tt.level >= logger.LevelDebug {
				assert.Contains(t, buf.String(), "New request:")
			}
		})
	}
}

func TestLoggerInterface(t *testing.T) {
	l := wslogger.NewLogger()
	var buf bytes.Buffer
	l.SetOutput(&buf)
	l.SetLevel(log.DEBUG) // Enable Debug logs

	l.Print("print")
	assert.Contains(t, buf.String(), "print")
	buf.Reset()

	l.Printf("printf %s", "test")
	assert.Contains(t, buf.String(), "printf test")
	buf.Reset()

	l.Printj(log.JSON{"key": "value"})
	assert.Contains(t, buf.String(), "map[key:value]")
	buf.Reset()

	l.Debug("debug")
	assert.Contains(t, buf.String(), "debug")
	buf.Reset()

	l.Debugf("debugf %s", "test")
	assert.Contains(t, buf.String(), "debugf test")
	buf.Reset()

	l.Debugj(log.JSON{"key": "value"})
	assert.Contains(t, buf.String(), "map[key:value]")
	buf.Reset()

	l.Info("info")
	assert.Contains(t, buf.String(), "info")
	buf.Reset()

	l.Infof("infof %s", "test")
	assert.Contains(t, buf.String(), "infof test")
	buf.Reset()

	l.Infoj(log.JSON{"key": "value"})
	assert.Contains(t, buf.String(), "map[key:value]")
	buf.Reset()

	l.Warn("warn")
	assert.Contains(t, buf.String(), "warn")
	buf.Reset()

	l.Warnf("warnf %s", "test")
	assert.Contains(t, buf.String(), "warnf test")
	buf.Reset()

	l.Warnj(log.JSON{"key": "value"})
	assert.Contains(t, buf.String(), "map[key:value]")
	buf.Reset()

	l.Error("error")
	assert.Contains(t, buf.String(), "error")
	buf.Reset()

	l.Errorf("errorf %s", "test")
	assert.Contains(t, buf.String(), "errorf test")
	buf.Reset()

	l.Errorj(log.JSON{"key": "value"})
	assert.Contains(t, buf.String(), "map[key:value]")
	buf.Reset()
}

func TestGettersSetters(t *testing.T) {
	l := wslogger.NewLogger()

	l.SetPrefix("[TEST]")
	assert.Equal(t, "[TEST]", l.Prefix())

	l.SetLevel(log.DEBUG)
	assert.Equal(t, log.DEBUG, l.Level())

	l.SetLevel(log.INFO)
	assert.Equal(t, log.INFO, l.Level())

	l.SetLevel(log.WARN)
	assert.Equal(t, log.WARN, l.Level())

	l.SetLevel(log.ERROR)
	assert.Equal(t, log.ERROR, l.Level())

	l.SetLevel(log.OFF) // Default case
	assert.Equal(t, log.INFO, l.Level())

	var buf bytes.Buffer
	l.SetOutput(&buf)
	assert.Equal(t, &buf, l.Output())

	l.SetHeader("header") // Does nothing but should be callable
}

func TestFatalAndPanic(t *testing.T) {
	// These exit or panic, so we can't easily test them in the same process without crashing.
	// We can skip them or run them in a subprocess if strict 100% is needed including these lines.
	// For now, let's assume we want to cover them if possible, but os.Exit(1) is hard to catch.
	// Panic can be recovered.

	l := wslogger.NewLogger()
	var buf bytes.Buffer
	l.SetOutput(&buf)

	assert.Panics(t, func() {
		l.Panic("panic")
	})
	assert.Contains(t, buf.String(), "panic")
	buf.Reset()

	assert.Panics(t, func() {
		l.Panicf("panicf %s", "test")
	})
	assert.Contains(t, buf.String(), "panicf test")
	buf.Reset()

	assert.Panics(t, func() {
		l.Panicj(log.JSON{"key": "value"})
	})
	assert.Contains(t, buf.String(), "map[key:value]")
}

func TestOutput_Nil(t *testing.T) {
	mock := &MockLogger{output: nil}
	l := wslogger.NewLogger(wslogger.WithLogger(&webserver.Logger{ILogger: mock}))
	assert.Nil(t, l.Output())
}

func TestLevel_Default(t *testing.T) {
	mock := &MockLogger{level: logger.Level(99)}
	l := wslogger.NewLogger(wslogger.WithLogger(&webserver.Logger{ILogger: mock}))
	assert.Equal(t, log.ERROR, l.Level())
}

func TestLevel_Panic(t *testing.T) {
	mock := &MockLogger{level: logger.LevelPanic}
	l := wslogger.NewLogger(wslogger.WithLogger(&webserver.Logger{ILogger: mock}))
	assert.Equal(t, log.ERROR, l.Level())
}

func TestSetLevel_Default(t *testing.T) {
	l := wslogger.NewLogger()
	l.SetLevel(log.Lvl(99))
	assert.Equal(t, log.INFO, l.Level())
}

func TestFatal(t *testing.T) {
	if os.Getenv("BE_CRASHER") == "1" {
		l := wslogger.NewLogger()
		l.Fatal("boom")
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestFatal")
	cmd.Env = append(os.Environ(), "BE_CRASHER=1")
	err := cmd.Run()
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		return
	}
	t.Fatalf("process ran with err %v, want exit status 1", err)
}

func TestFatalf(t *testing.T) {
	if os.Getenv("BE_CRASHER") == "1" {
		l := wslogger.NewLogger()
		l.Fatalf("boom %s", "formatted")
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestFatalf")
	cmd.Env = append(os.Environ(), "BE_CRASHER=1")
	err := cmd.Run()
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		return
	}
	t.Fatalf("process ran with err %v, want exit status 1", err)
}

func TestFatalj(t *testing.T) {
	if os.Getenv("BE_CRASHER") == "1" {
		l := wslogger.NewLogger()
		l.Fatalj(log.JSON{"key": "value"})
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestFatalj")
	cmd.Env = append(os.Environ(), "BE_CRASHER=1")
	err := cmd.Run()
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		return
	}
	t.Fatalf("process ran with err %v, want exit status 1", err)
}

func TestNewLogger_WithJSONLogger(t *testing.T) {
	l := wslogger.NewLogger(wslogger.WithJSONLogger())
	assert.NotNil(t, l)
	assert.Equal(t, "[ECHO]", l.Prefix())
}

func TestMiddleware_WithJSONLogger(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	l := wslogger.NewLogger(
		wslogger.WithJSONLogger(),
		wslogger.WithLogProtocol(),
		wslogger.WithLogMethod(),
		wslogger.WithLogURI(),
		wslogger.WithLogStatus(),
		wslogger.WithLogLatency(),
	)

	var buf bytes.Buffer
	l.SetOutput(&buf)

	h := l.ToMiddleware()(func(c webserver.Context) error {
		return c.String(http.StatusOK, "json test")
	})

	err := h(c)
	assert.NoError(t, err)

	// Verify the output is valid JSON
	var parsed map[string]interface{}
	assert.NoError(t, json.Unmarshal(buf.Bytes(), &parsed))
	assert.Contains(t, parsed["message"], "New request:")
	assert.Contains(t, parsed["message"], "Method=GET")
}

