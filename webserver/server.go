package webserver

import (
	"context"
	"time"

	"github.com/GabrielNunesIT/go-libs/logger"
	"github.com/labstack/echo-contrib/echoprometheus"
	"github.com/labstack/echo-contrib/pprof"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// Context is the context for the web server.
type Context interface {
	echo.Context
}

// HandlerFunc is the handler function for the web server.
type HandlerFunc func(Context) error

// MiddlewareFunc is the middleware function for the web server.
type MiddlewareFunc func(HandlerFunc) HandlerFunc

// Route is a web route representation.
type Route struct {
	Method string `json:"method"`
	Path   string `json:"path"`
	Name   string `json:"name"`
}

// CORSConfig defines the configuration for Cross-Origin Resource Sharing (CORS).
type CORSConfig struct {
	AllowOrigins []string
	AllowMethods []string
	AllowHeaders []string
}

// WebServer is the web server.
type WebServer struct {
	framework *echo.Echo
	address   string
}

// Option defines a configuration option for the WebServer.
type Option func(*WebServer)

// New creates a new WebServer with the given options.
func New(opts ...Option) *WebServer {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	server := &WebServer{
		framework: e,
		address:   ":0", // Random address
	}

	for _, opt := range opts {
		opt(server)
	}

	return server
}

// WithAddress sets the address for the WebServer.
func WithAddress(address string) Option {
	return func(server *WebServer) {
		server.address = address
	}
}

// WithReadTimeout sets the read timeout for the WebServer.
func WithReadTimeout(timeout time.Duration) Option {
	return func(server *WebServer) {
		server.framework.Server.ReadTimeout = timeout
	}
}

// WithWriteTimeout sets the write timeout for the WebServer.
func WithWriteTimeout(timeout time.Duration) Option {
	return func(server *WebServer) {
		server.framework.Server.WriteTimeout = timeout
	}
}

// WithMiddleware adds middlewares to the WebServer.
func WithMiddleware(middlewares ...MiddlewareFunc) Option {
	return func(server *WebServer) {
		for _, m := range middlewares {
			server.framework.Use(wrapMiddleware(m))
		}
	}
}

// WithRecovery adds the recovery middleware to the WebServer.
func WithRecovery() Option {
	return func(server *WebServer) {
		server.framework.Use(middleware.Recover())
	}
}

// WithCORS adds the CORS middleware to the WebServer.
func WithCORS(config CORSConfig) Option {
	return func(server *WebServer) {
		server.framework.Use(middleware.CORSWithConfig(middleware.CORSConfig{
			AllowOrigins: config.AllowOrigins,
			AllowMethods: config.AllowMethods,
			AllowHeaders: config.AllowHeaders,
		}))
	}
}

// WithRequestID adds the request ID middleware to the WebServer. Uses "UNIQUE_ID" header.
func WithRequestID() Option {
	return func(server *WebServer) {
		server.framework.Use(middleware.RequestIDWithConfig(middleware.RequestIDConfig{
			Skipper: func(c echo.Context) bool {
				return c.Request().Header.Get("UNIQUE_ID") != ""
			},
			TargetHeader: "UNIQUE_ID",
		}))
	}
}

// WithCustomRequestID adds a custom request ID middleware to the WebServer.
func WithCustomRequestID(header string) Option {
	return func(server *WebServer) {
		server.framework.Use(middleware.RequestIDWithConfig(middleware.RequestIDConfig{
			Skipper: func(c echo.Context) bool {
				return c.Request().Header.Get(header) != ""
			},
			TargetHeader: header,
		}))
	}
}

// WithBodyLimit sets a body size limit for incoming requests.
func WithBodyLimit(limit string) Option {
	return func(server *WebServer) {
		server.framework.Use(middleware.BodyLimit(limit))
	}
}

// WithContextTimeout sets a context timeout for incoming requests.
func WithContextTimeout(timeout time.Duration) Option {
	return func(server *WebServer) {
		server.framework.Use(middleware.ContextTimeout(timeout))
	}
}

// WithCustomMiddleware adds a custom middleware to the WebServer.
func WithCustomMiddleware(mw MiddlewareFunc) Option {
	return func(server *WebServer) {
		server.framework.Use(wrapMiddleware(mw))
	}
}

// WithPrometheus adds Prometheus metrics middleware to the WebServer.
func WithPrometheus(title, route string) Option {
	return func(server *WebServer) {
		server.framework.Use(echoprometheus.NewMiddleware(title))
		server.framework.GET(route, echoprometheus.NewHandler())
	}
}

// WithPprof adds pprof middleware to the WebServer.
func WithPprof() Option {
	return func(server *WebServer) {
		pprof.Register(server.framework)
	}
}

// WithLogger sets the logger for the WebServer.
func WithLogger(l logger.ILogger) Option {
	return func(server *WebServer) {
		server.framework.Logger = &Logger{ILogger: l}
	}
}

// StartHTTP starts the WebServer.
func (server *WebServer) StartHTTP() error {
	//nolint:wrapcheck // we want to return the error from echo directly
	return server.framework.Start(server.address)
}

// StartHTTPS starts the WebServer with TLS.
func (server *WebServer) StartHTTPS(certFile, keyFile string) error {
	//nolint:wrapcheck // we want to return the error from echo directly
	return server.framework.StartTLS(server.address, certFile, keyFile)
}

// Shutdown shuts down the WebServer gracefully.
func (server *WebServer) Shutdown(ctx context.Context) error {
	//nolint:wrapcheck // we want to return the error from echo directly
	return server.framework.Shutdown(ctx)
}

// CONNECT registers a new CONNECT route.
func (server *WebServer) CONNECT(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) *Route {
	r := server.framework.CONNECT(path, wrapHandler(handler), wrapMiddlewares(middlewares)...)
	return &Route{
		Method: r.Method,
		Path:   r.Path,
		Name:   r.Name,
	}
}

// DELETE registers a new DELETE route.
func (server *WebServer) DELETE(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) *Route {
	r := server.framework.DELETE(path, wrapHandler(handler), wrapMiddlewares(middlewares)...)
	return &Route{
		Method: r.Method,
		Path:   r.Path,
		Name:   r.Name,
	}
}

// GET registers a new GET route.
func (server *WebServer) GET(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) *Route {
	r := server.framework.GET(path, wrapHandler(handler), wrapMiddlewares(middlewares)...)
	return &Route{
		Method: r.Method,
		Path:   r.Path,
		Name:   r.Name,
	}
}

// HEAD registers a new HEAD route.
func (server *WebServer) HEAD(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) *Route {
	r := server.framework.HEAD(path, wrapHandler(handler), wrapMiddlewares(middlewares)...)
	return &Route{
		Method: r.Method,
		Path:   r.Path,
		Name:   r.Name,
	}
}

// OPTIONS registers a new OPTIONS route.
func (server *WebServer) OPTIONS(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) *Route {
	r := server.framework.OPTIONS(path, wrapHandler(handler), wrapMiddlewares(middlewares)...)
	return &Route{
		Method: r.Method,
		Path:   r.Path,
		Name:   r.Name,
	}
}

// PATCH registers a new PATCH route.
func (server *WebServer) PATCH(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) *Route {
	r := server.framework.PATCH(path, wrapHandler(handler), wrapMiddlewares(middlewares)...)
	return &Route{
		Method: r.Method,
		Path:   r.Path,
		Name:   r.Name,
	}
}

// POST registers a new POST route.
func (server *WebServer) POST(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) *Route {
	r := server.framework.POST(path, wrapHandler(handler), wrapMiddlewares(middlewares)...)
	return &Route{
		Method: r.Method,
		Path:   r.Path,
		Name:   r.Name,
	}
}

// PUT registers a new PUT route.
func (server *WebServer) PUT(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) *Route {
	r := server.framework.PUT(path, wrapHandler(handler), wrapMiddlewares(middlewares)...)
	return &Route{
		Method: r.Method,
		Path:   r.Path,
		Name:   r.Name,
	}
}

// TRACE registers a new TRACE route.
func (server *WebServer) TRACE(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) *Route {
	r := server.framework.TRACE(path, wrapHandler(handler), wrapMiddlewares(middlewares)...)
	return &Route{
		Method: r.Method,
		Path:   r.Path,
		Name:   r.Name,
	}
}

// ANY registers a new route for all HTTP methods.
func (server *WebServer) ANY(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) []*Route {
	routes := make([]*Route, len(methods))
	for i, m := range methods {
		r := server.framework.Add(m, path, wrapHandler(handler), wrapMiddlewares(middlewares)...)
		routes[i] = &Route{
			Method: r.Method,
			Path:   r.Path,
			Name:   r.Name,
		}
	}
	return routes
}

// Group creates a new route group.
func (server *WebServer) Group(prefix string, middlewares ...MiddlewareFunc) *Group {
	return &Group{group: server.framework.Group(prefix, wrapMiddlewares(middlewares)...)}
}
