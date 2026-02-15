package metrics_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/GabrielNunesIT/go-libs/metrics"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHTTPMetrics(t *testing.T) {
	t.Parallel()

	reg := metrics.New(metrics.WithNamespace("app"))
	m := metrics.NewHTTPMetrics(reg)

	assert.NotNil(t, m.RequestsTotal())
	assert.NotNil(t, m.RequestDuration())
	assert.NotNil(t, m.RequestsInFlight())
}

func TestHTTPMiddleware(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		method         string
		path           string
		handlerStatus  int
		handlerBody    string
		expectedStatus string
	}{
		{
			name:           "GET 200",
			method:         http.MethodGet,
			path:           "/api/users",
			handlerStatus:  http.StatusOK,
			handlerBody:    "ok",
			expectedStatus: "200",
		},
		{
			name:           "POST 201",
			method:         http.MethodPost,
			path:           "/api/users",
			handlerStatus:  http.StatusCreated,
			handlerBody:    "created",
			expectedStatus: "201",
		},
		{
			name:           "GET 404",
			method:         http.MethodGet,
			path:           "/api/missing",
			handlerStatus:  http.StatusNotFound,
			handlerBody:    "not found",
			expectedStatus: "404",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			reg := metrics.New()
			m := metrics.NewHTTPMetrics(reg)

			inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.handlerStatus)
				_, _ = w.Write([]byte(tt.handlerBody))
			})

			handler := m.Middleware(inner)

			req := httptest.NewRequest(tt.method, tt.path, nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			assert.Equal(t, tt.handlerStatus, rec.Code)
			assert.Equal(t, tt.handlerBody, rec.Body.String())

			// Verify metrics were recorded
			families, err := reg.PrometheusRegistry().Gather()
			require.NoError(t, err)

			counterFam := findFamily(families, "http_requests_total")
			require.NotNil(t, counterFam)
			assert.Len(t, counterFam.GetMetric(), 1)

			metric := counterFam.GetMetric()[0]
			assert.InDelta(t, 1.0, metric.GetCounter().GetValue(), 0.001)

			// Verify labels
			labelMap := labelPairs(metric)
			assert.Equal(t, tt.method, labelMap["method"])
			assert.Equal(t, tt.path, labelMap["path"])
			assert.Equal(t, tt.expectedStatus, labelMap["status"])

			// Verify histogram was observed
			histFam := findFamily(families, "http_request_duration_seconds")
			require.NotNil(t, histFam)
			assert.Equal(t, uint64(1), histFam.GetMetric()[0].GetHistogram().GetSampleCount())

			// In-flight should be 0 after request completes
			gaugeFam := findFamily(families, "http_requests_in_flight")
			require.NotNil(t, gaugeFam)
			assert.InDelta(t, 0.0, gaugeFam.GetMetric()[0].GetGauge().GetValue(), 0.001)
		})
	}
}

func TestHTTPMiddlewareDefaultStatus(t *testing.T) {
	t.Parallel()

	reg := metrics.New()
	m := metrics.NewHTTPMetrics(reg)

	// Handler that writes body without explicit WriteHeader → defaults to 200
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("implicit 200"))
	})

	handler := m.Middleware(inner)

	req := httptest.NewRequest(http.MethodGet, "/implicit", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	families, err := reg.PrometheusRegistry().Gather()
	require.NoError(t, err)

	counterFam := findFamily(families, "http_requests_total")
	require.NotNil(t, counterFam)

	labelMap := labelPairs(counterFam.GetMetric()[0])
	assert.Equal(t, "200", labelMap["status"])
}

func TestWithHTTPBuckets(t *testing.T) {
	t.Parallel()

	customBuckets := []float64{0.01, 0.1, 1.0}
	reg := metrics.New()
	m := metrics.NewHTTPMetrics(reg, metrics.WithHTTPBuckets(customBuckets))

	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := m.Middleware(inner)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	families, err := reg.PrometheusRegistry().Gather()
	require.NoError(t, err)

	histFam := findFamily(families, "http_request_duration_seconds")
	require.NotNil(t, histFam)

	// Custom buckets: 0.01, 0.1, 1.0 → 3 explicit + 1 (+Inf) = 4 buckets
	hist := histFam.GetMetric()[0].GetHistogram()
	assert.Len(t, hist.GetBucket(), len(customBuckets))
}

func TestHTTPMultipleRequests(t *testing.T) {
	t.Parallel()

	reg := metrics.New()
	m := metrics.NewHTTPMetrics(reg)

	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := m.Middleware(inner)

	for range 5 {
		req := httptest.NewRequest(http.MethodGet, "/count", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}

	families, err := reg.PrometheusRegistry().Gather()
	require.NoError(t, err)

	counterFam := findFamily(families, "http_requests_total")
	require.NotNil(t, counterFam)
	assert.InDelta(t, 5.0, counterFam.GetMetric()[0].GetCounter().GetValue(), 0.001)
}

func labelPairs(m *dto.Metric) map[string]string {
	result := make(map[string]string)
	for _, lp := range m.GetLabel() {
		result[lp.GetName()] = lp.GetValue()
	}

	return result
}
