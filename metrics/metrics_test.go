package metrics_test

import (
	"testing"

	"github.com/GabrielNunesIT/go-libs/metrics"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func collectMetricFamilies(t *testing.T, reg *metrics.Registry) []*dto.MetricFamily {
	t.Helper()

	families, err := reg.PrometheusRegistry().Gather()
	require.NoError(t, err)

	return families
}

func findFamily(families []*dto.MetricFamily, name string) *dto.MetricFamily {
	for _, f := range families {
		if f.GetName() == name {
			return f
		}
	}

	return nil
}

func TestNew(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		opts []metrics.Option
	}{
		{
			name: "default registry",
			opts: nil,
		},
		{
			name: "with namespace",
			opts: []metrics.Option{metrics.WithNamespace("myapp")},
		},
		{
			name: "with namespace and subsystem",
			opts: []metrics.Option{
				metrics.WithNamespace("myapp"),
				metrics.WithSubsystem("api"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			reg := metrics.New(tt.opts...)
			assert.NotNil(t, reg)
			assert.NotNil(t, reg.PrometheusRegistry())
		})
	}
}

func TestWithProcessCollector(t *testing.T) {
	t.Parallel()

	reg := metrics.New(metrics.WithProcessCollector())
	families := collectMetricFamilies(t, reg)

	assert.NotEmpty(t, families, "process collector should produce metrics")
	assert.NotNil(t, findFamily(families, "process_open_fds"))
}

func TestWithGoCollector(t *testing.T) {
	t.Parallel()

	reg := metrics.New(metrics.WithGoCollector())
	families := collectMetricFamilies(t, reg)

	assert.NotEmpty(t, families, "go collector should produce metrics")
	assert.NotNil(t, findFamily(families, "go_goroutines"))
}

func TestNewCounter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		namespace      string
		subsystem      string
		metricName     string
		expectedFamily string
	}{
		{
			name:           "without prefix",
			metricName:     "test_total",
			expectedFamily: "test_total",
		},
		{
			name:           "with namespace",
			namespace:      "myapp",
			metricName:     "test_total",
			expectedFamily: "myapp_test_total",
		},
		{
			name:           "with namespace and subsystem",
			namespace:      "myapp",
			subsystem:      "api",
			metricName:     "test_total",
			expectedFamily: "myapp_api_test_total",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var opts []metrics.Option
			if tt.namespace != "" {
				opts = append(opts, metrics.WithNamespace(tt.namespace))
			}

			if tt.subsystem != "" {
				opts = append(opts, metrics.WithSubsystem(tt.subsystem))
			}

			reg := metrics.New(opts...)

			c := reg.NewCounter(tt.metricName, "A test counter")
			c.Inc()

			families := collectMetricFamilies(t, reg)

			fam := findFamily(families, tt.expectedFamily)
			require.NotNil(t, fam, "expected family %s", tt.expectedFamily)
			assert.Equal(t, dto.MetricType_COUNTER, fam.GetType())
			assert.InDelta(t, 1.0, fam.GetMetric()[0].GetCounter().GetValue(), 0.001)
		})
	}
}

func TestNewCounterVec(t *testing.T) {
	t.Parallel()

	reg := metrics.New(metrics.WithNamespace("app"))
	cv := reg.NewCounterVec("events_total", "Event count", []string{"type"})

	cv.WithLabelValues("click").Inc()
	cv.WithLabelValues("click").Inc()
	cv.WithLabelValues("scroll").Inc()

	families := collectMetricFamilies(t, reg)
	fam := findFamily(families, "app_events_total")
	require.NotNil(t, fam)
	assert.Len(t, fam.GetMetric(), 2) //nolint:mnd // two distinct label sets
}

func TestNewGauge(t *testing.T) {
	t.Parallel()

	reg := metrics.New()

	g := reg.NewGauge("temperature", "Current temp")
	g.Set(42.5)

	families := collectMetricFamilies(t, reg)
	fam := findFamily(families, "temperature")
	require.NotNil(t, fam)
	assert.Equal(t, dto.MetricType_GAUGE, fam.GetType())
	assert.InDelta(t, 42.5, fam.GetMetric()[0].GetGauge().GetValue(), 0.001)
}

func TestNewGaugeVec(t *testing.T) {
	t.Parallel()

	reg := metrics.New()
	gv := reg.NewGaugeVec("pool_size", "Pool size", []string{"pool"})

	gv.WithLabelValues("workers").Set(10)

	families := collectMetricFamilies(t, reg)
	fam := findFamily(families, "pool_size")
	require.NotNil(t, fam)
	assert.InDelta(t, 10.0, fam.GetMetric()[0].GetGauge().GetValue(), 0.001)
}

func TestNewHistogram(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		buckets []float64
	}{
		{
			name:    "default buckets",
			buckets: nil,
		},
		{
			name:    "custom buckets",
			buckets: []float64{0.1, 0.5, 1.0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			reg := metrics.New()
			h := reg.NewHistogram("latency_seconds", "Latency", tt.buckets)

			h.Observe(0.25)
			h.Observe(0.75)

			families := collectMetricFamilies(t, reg)
			fam := findFamily(families, "latency_seconds")
			require.NotNil(t, fam)
			assert.Equal(t, dto.MetricType_HISTOGRAM, fam.GetType())
			assert.Equal(t, uint64(2), fam.GetMetric()[0].GetHistogram().GetSampleCount())
		})
	}
}

func TestNewHistogramVec(t *testing.T) {
	t.Parallel()

	reg := metrics.New()
	hv := reg.NewHistogramVec("request_size_bytes", "Req size", []string{"endpoint"}, nil)

	hv.WithLabelValues("/api").Observe(512)

	families := collectMetricFamilies(t, reg)
	fam := findFamily(families, "request_size_bytes")
	require.NotNil(t, fam)
	assert.Equal(t, uint64(1), fam.GetMetric()[0].GetHistogram().GetSampleCount())
}

func TestHandler(t *testing.T) {
	t.Parallel()

	reg := metrics.New()
	reg.NewCounter("handler_check", "check")

	handler := reg.Handler()
	assert.NotNil(t, handler)

	standaloneHandler := metrics.Handler(reg)
	assert.NotNil(t, standaloneHandler)
}

func TestDuplicateRegistrationPanics(t *testing.T) {
	t.Parallel()

	reg := metrics.New()
	reg.NewCounter("dup_counter", "first")

	assert.Panics(t, func() {
		reg.NewCounter("dup_counter", "second")
	})
}

func TestDefaultHistogramBuckets(t *testing.T) {
	t.Parallel()

	assert.Equal(t, prometheus.DefBuckets, []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10})
	assert.Equal(t, metrics.DefaultHistogramBuckets, []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10})
}
