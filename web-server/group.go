// Package webserver provides a web server implementation.
package webserver

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

const (
	PROPFIND = "PROPFIND"
	REPORT   = "REPORT"
)

var methods = [...]string{
	http.MethodConnect,
	http.MethodDelete,
	http.MethodGet,
	http.MethodHead,
	http.MethodOptions,
	http.MethodPatch,
	http.MethodPost,
	PROPFIND,
	http.MethodPut,
	http.MethodTrace,
	REPORT,
}

// Group represents a route group.
type Group struct {
	group *echo.Group
}

// CONNECT registers a new CONNECT route in the group.
func (g *Group) CONNECT(path string, handler HandlerFunc, middleware ...MiddlewareFunc) *Route {
	r := g.group.CONNECT(path, wrapHandler(handler), wrapMiddlewares(middleware)...)
	return &Route{
		Method: r.Method,
		Path:   r.Path,
		Name:   r.Name,
	}
}

// DELETE registers a new DELETE route in the group.
func (g *Group) DELETE(path string, handler HandlerFunc, middleware ...MiddlewareFunc) *Route {
	r := g.group.DELETE(path, wrapHandler(handler), wrapMiddlewares(middleware)...)
	return &Route{
		Method: r.Method,
		Path:   r.Path,
		Name:   r.Name,
	}
}

// GET registers a new GET route in the group.
func (g *Group) GET(path string, handler HandlerFunc, middleware ...MiddlewareFunc) *Route {
	r := g.group.GET(path, wrapHandler(handler), wrapMiddlewares(middleware)...)
	return &Route{
		Method: r.Method,
		Path:   r.Path,
		Name:   r.Name,
	}
}

// HEAD registers a new HEAD route in the group.
func (g *Group) HEAD(path string, handler HandlerFunc, middleware ...MiddlewareFunc) *Route {
	r := g.group.HEAD(path, wrapHandler(handler), wrapMiddlewares(middleware)...)
	return &Route{
		Method: r.Method,
		Path:   r.Path,
		Name:   r.Name,
	}
}

// OPTIONS registers a new OPTIONS route in the group.
func (g *Group) OPTIONS(path string, handler HandlerFunc, middleware ...MiddlewareFunc) *Route {
	r := g.group.OPTIONS(path, wrapHandler(handler), wrapMiddlewares(middleware)...)
	return &Route{
		Method: r.Method,
		Path:   r.Path,
		Name:   r.Name,
	}
}

// PATCH registers a new PATCH route in the group.
func (g *Group) PATCH(path string, handler HandlerFunc, middleware ...MiddlewareFunc) *Route {
	r := g.group.PATCH(path, wrapHandler(handler), wrapMiddlewares(middleware)...)
	return &Route{
		Method: r.Method,
		Path:   r.Path,
		Name:   r.Name,
	}
}

// POST registers a new POST route in the group.
func (g *Group) POST(path string, handler HandlerFunc, middleware ...MiddlewareFunc) *Route {
	r := g.group.POST(path, wrapHandler(handler), wrapMiddlewares(middleware)...)
	return &Route{
		Method: r.Method,
		Path:   r.Path,
		Name:   r.Name,
	}
}

// PUT registers a new PUT route in the group.
func (g *Group) PUT(path string, handler HandlerFunc, middleware ...MiddlewareFunc) *Route {
	r := g.group.PUT(path, wrapHandler(handler), wrapMiddlewares(middleware)...)
	return &Route{
		Method: r.Method,
		Path:   r.Path,
		Name:   r.Name,
	}
}

// TRACE registers a new TRACE route in the group.
func (g *Group) TRACE(path string, handler HandlerFunc, middleware ...MiddlewareFunc) *Route {
	r := g.group.TRACE(path, wrapHandler(handler), wrapMiddlewares(middleware)...)
	return &Route{
		Method: r.Method,
		Path:   r.Path,
		Name:   r.Name,
	}
}

// ANY registers a new route for all HTTP methods in the group.
func (g *Group) ANY(path string, handler HandlerFunc, middleware ...MiddlewareFunc) []*Route {
	routes := make([]*Route, len(methods))
	for i, m := range methods {
		r := g.group.Add(m, path, wrapHandler(handler), wrapMiddlewares(middleware)...)
		routes[i] = &Route{
			Method: r.Method,
			Path:   r.Path,
			Name:   r.Name,
		}
	}
	return routes
}

// Group creates a new sub-group.
func (g *Group) Group(prefix string, middleware ...MiddlewareFunc) *Group {
	return &Group{group: g.group.Group(prefix, wrapMiddlewares(middleware)...)}
}

// Use adds middleware to the group.
func (g *Group) Use(middleware ...MiddlewareFunc) {
	g.group.Use(wrapMiddlewares(middleware)...)
}
