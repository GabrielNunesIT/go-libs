package webserver

import "github.com/labstack/echo/v4"

func wrapHandler(h HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		return h(c.(Context))
	}
}

func wrapMiddleware(m MiddlewareFunc) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			return m(func(ctx Context) error {
				return next(ctx)
			})(c)
		}
	}
}

func wrapMiddlewares(middlewares []MiddlewareFunc) []echo.MiddlewareFunc {
	result := make([]echo.MiddlewareFunc, len(middlewares))
	for i, m := range middlewares {
		result[i] = wrapMiddleware(m)
	}
	return result
}
