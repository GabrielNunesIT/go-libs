package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// HTTPMetrics provides predefined Prometheus metrics for HTTP servers:
// request counter, request duration histogram, and in-flight request gauge.
type HTTPMetrics struct {
	requestsTotal    *prometheus.CounterVec
	requestDuration  *prometheus.HistogramVec
	requestsInFlight prometheus.Gauge
	buckets          []float64
}

// HTTPOption configures HTTPMetrics.
type HTTPOption func(*HTTPMetrics)

// WithHTTPBuckets overrides the default histogram buckets for request
// duration tracking.
func WithHTTPBuckets(buckets []float64) HTTPOption {
	return func(m *HTTPMetrics) {
		m.buckets = buckets
	}
}

// NewHTTPMetrics creates and registers a predefined set of HTTP metrics on the
// given Registry. The following metrics are created:
//
//   - http_requests_total (counter vec: method, path, status)
//   - http_request_duration_seconds (histogram vec: method, path, status)
//   - http_requests_in_flight (gauge)
func NewHTTPMetrics(reg *Registry, opts ...HTTPOption) *HTTPMetrics {
	httpMetrics := &HTTPMetrics{
		buckets: DefaultHistogramBuckets,
	}

	for _, opt := range opts {
		opt(httpMetrics)
	}

	labels := []string{"method", "path", "status"}

	httpMetrics.requestsTotal = reg.NewCounterVec(
		"http_requests_total",
		"Total number of HTTP requests processed.",
		labels,
	)
	httpMetrics.requestDuration = reg.NewHistogramVec(
		"http_request_duration_seconds",
		"Duration of HTTP requests in seconds.",
		labels,
		httpMetrics.buckets,
	)
	httpMetrics.requestsInFlight = reg.NewGauge(
		"http_requests_in_flight",
		"Number of HTTP requests currently being processed.",
	)

	return httpMetrics
}

// RequestsTotal returns the underlying counter vec so callers can use it
// directly when the middleware approach is not suitable.
func (m *HTTPMetrics) RequestsTotal() *prometheus.CounterVec {
	return m.requestsTotal
}

// RequestDuration returns the underlying histogram vec.
func (m *HTTPMetrics) RequestDuration() *prometheus.HistogramVec {
	return m.requestDuration
}

// RequestsInFlight returns the underlying in-flight gauge.
//
//nolint:ireturn // prometheus.Gauge has no exported concrete type
func (m *HTTPMetrics) RequestsInFlight() prometheus.Gauge {
	return m.requestsInFlight
}

// responseWriter wraps http.ResponseWriter to capture the status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
}

// WriteHeader captures the status code before delegating.
func (rw *responseWriter) WriteHeader(code int) {
	if !rw.written {
		rw.statusCode = code
		rw.written = true
	}

	rw.ResponseWriter.WriteHeader(code)
}

// Write ensures the status code is captured on first write.
func (rw *responseWriter) Write(data []byte) (int, error) {
	if !rw.written {
		rw.written = true
	}

	//nolint:wrapcheck // transparent proxy
	return rw.ResponseWriter.Write(data)
}

// Middleware returns an http.Handler that wraps next with Prometheus
// instrumentation for request count, duration, and in-flight tracking.
func (m *HTTPMetrics) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
		m.requestsInFlight.Inc()
		defer m.requestsInFlight.Dec()

		start := time.Now()
		rw := newResponseWriter(writer)

		next.ServeHTTP(rw, req)

		statusCode := strconv.Itoa(rw.statusCode)
		elapsed := time.Since(start).Seconds()

		m.requestsTotal.WithLabelValues(req.Method, req.URL.Path, statusCode).Inc()
		m.requestDuration.WithLabelValues(req.Method, req.URL.Path, statusCode).Observe(elapsed)
	})
}
