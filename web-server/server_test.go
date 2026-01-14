package webserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewWebServer(t *testing.T) {
	ws := New(
		WithAddress(":9090"),
		WithMiddleware(func(next HandlerFunc) HandlerFunc {
			return func(c Context) error {
				c.Response().Header().Set("X-Test-Middleware", "true")
				return next(c)
			}
		}),
	)

	assert.NotNil(t, ws)
	assert.Equal(t, ":9090", ws.address)
	assert.NotNil(t, ws.framework)
}

func TestRouteRegistration(t *testing.T) {
	ws := New(
		WithMiddleware(func(next HandlerFunc) HandlerFunc {
			return func(c Context) error {
				c.Response().Header().Set("X-Test-Middleware", "true")
				return next(c)
			}
		}),
	)

	ws.GET("/test", func(c Context) error {
		return c.String(http.StatusOK, "test")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	ws.framework.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "test", rec.Body.String())
	assert.Equal(t, "true", rec.Header().Get("X-Test-Middleware"))
}

func TestWithAddress(t *testing.T) {
	wsAddr := New(WithAddress("localhost:8081"))
	assert.Equal(t, "localhost:8081", wsAddr.address)
}

func TestRecovery(t *testing.T) {
	wsRecovery := New(WithRecovery())
	wsRecovery.GET("/panic", func(c Context) error {
		panic("oops")
	})
	reqPanic := httptest.NewRequest(http.MethodGet, "/panic", nil)
	recPanic := httptest.NewRecorder()
	wsRecovery.framework.ServeHTTP(recPanic, reqPanic)
	assert.Equal(t, http.StatusInternalServerError, recPanic.Code)
}

func TestCORS(t *testing.T) {
	wsCORS := New(WithCORS(CORSConfig{
		AllowOrigins: []string{"https://example.com"},
		AllowMethods: []string{http.MethodGet},
	}))
	wsCORS.GET("/cors", func(c Context) error {
		return c.String(http.StatusOK, "cors")
	})
	reqCORS := httptest.NewRequest(http.MethodOptions, "/cors", nil)
	reqCORS.Header.Set("Origin", "https://example.com")
	reqCORS.Header.Set("Access-Control-Request-Method", http.MethodGet)
	recCORS := httptest.NewRecorder()
	wsCORS.framework.ServeHTTP(recCORS, reqCORS)

	assert.Equal(t, http.StatusNoContent, recCORS.Code)
	assert.Equal(t, "https://example.com", recCORS.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, http.MethodGet, recCORS.Header().Get("Access-Control-Allow-Methods"))
}

func TestRequestID(t *testing.T) {
	ws := New(WithRequestID())
	ws.GET("/request-id", func(c Context) error {
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/request-id", nil)
	rec := httptest.NewRecorder()
	ws.framework.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.NotEmpty(t, rec.Header().Get("UNIQUE_ID"))
}

func TestCustomRequestID(t *testing.T) {
	ws := New(WithCustomRequestID("X-Custom-ID"))
	ws.GET("/custom-request-id", func(c Context) error {
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/custom-request-id", nil)
	rec := httptest.NewRecorder()
	ws.framework.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.NotEmpty(t, rec.Header().Get("X-Custom-ID"))
}

func TestWithCustomMiddleware(t *testing.T) {
	ws := New(WithCustomMiddleware(func(next HandlerFunc) HandlerFunc {
		return func(c Context) error {
			c.Response().Header().Set("X-Custom-MW", "true")
			return next(c)
		}
	}))
	ws.GET("/custom-mw", func(c Context) error {
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/custom-mw", nil)
	rec := httptest.NewRecorder()
	ws.framework.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "true", rec.Header().Get("X-Custom-MW"))
}

func TestHTTPMethods(t *testing.T) {
	ws := New()

	ws.POST("/post", func(c Context) error {
		return c.String(http.StatusCreated, "post")
	})
	ws.PUT("/put", func(c Context) error {
		return c.String(http.StatusOK, "put")
	})
	ws.DELETE("/delete", func(c Context) error {
		return c.String(http.StatusNoContent, "")
	})

	// Test POST
	reqPost := httptest.NewRequest(http.MethodPost, "/post", nil)
	recPost := httptest.NewRecorder()
	ws.framework.ServeHTTP(recPost, reqPost)
	assert.Equal(t, http.StatusCreated, recPost.Code)
	assert.Equal(t, "post", recPost.Body.String())

	// Test PUT
	reqPut := httptest.NewRequest(http.MethodPut, "/put", nil)
	recPut := httptest.NewRecorder()
	ws.framework.ServeHTTP(recPut, reqPut)
	assert.Equal(t, http.StatusOK, recPut.Code)
	assert.Equal(t, "put", recPut.Body.String())

	// Test DELETE
	reqDel := httptest.NewRequest(http.MethodDelete, "/delete", nil)
	recDel := httptest.NewRecorder()
	ws.framework.ServeHTTP(recDel, reqDel)
	assert.Equal(t, http.StatusNoContent, recDel.Code)
}

func TestGroupMethods(t *testing.T) {
	ws := New()
	g := ws.Group("/api")

	g.Use(func(next HandlerFunc) HandlerFunc {
		return func(c Context) error {
			c.Response().Header().Set("X-Group-MW", "true")
			return next(c)
		}
	})

	g.GET("/get", func(c Context) error {
		return c.String(http.StatusOK, "get")
	})
	g.POST("/post", func(c Context) error {
		return c.String(http.StatusCreated, "post")
	})
	g.PUT("/put", func(c Context) error {
		return c.String(http.StatusOK, "put")
	})
	g.DELETE("/delete", func(c Context) error {
		return c.String(http.StatusNoContent, "")
	})

	// Test Group GET
	reqGet := httptest.NewRequest(http.MethodGet, "/api/get", nil)
	recGet := httptest.NewRecorder()
	ws.framework.ServeHTTP(recGet, reqGet)
	assert.Equal(t, http.StatusOK, recGet.Code)
	assert.Equal(t, "get", recGet.Body.String())
	assert.Equal(t, "true", recGet.Header().Get("X-Group-MW"))

	// Test Group POST
	reqPost := httptest.NewRequest(http.MethodPost, "/api/post", nil)
	recPost := httptest.NewRecorder()
	ws.framework.ServeHTTP(recPost, reqPost)
	assert.Equal(t, http.StatusCreated, recPost.Code)

	// Test Group PUT
	reqPut := httptest.NewRequest(http.MethodPut, "/api/put", nil)
	recPut := httptest.NewRecorder()
	ws.framework.ServeHTTP(recPut, reqPut)
	assert.Equal(t, http.StatusOK, recPut.Code)

	// Test Group DELETE
	reqDel := httptest.NewRequest(http.MethodDelete, "/api/delete", nil)
	recDel := httptest.NewRecorder()
	ws.framework.ServeHTTP(recDel, reqDel)
	assert.Equal(t, http.StatusNoContent, recDel.Code)
}

func TestSubGroup(t *testing.T) {
	ws := New()
	g := ws.Group("/v1")
	sub := g.Group("/users")

	sub.GET("", func(c Context) error {
		return c.String(http.StatusOK, "users")
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/users", nil)
	rec := httptest.NewRecorder()
	ws.framework.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "users", rec.Body.String())
}

func TestStartShutdown(t *testing.T) {
	// This is tricky to test because Start blocks.
	// We can try to start in a goroutine and then shutdown.
	ws := New(WithAddress(":0"))

	go func() {
		_ = ws.StartHTTP()
	}()

	// Give it a moment to start? Or just call Shutdown immediately.
	// Shutdown requires a context.
	err := ws.Shutdown(context.Background())
	assert.NoError(t, err)
}
