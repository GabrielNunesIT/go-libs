// Package metrics provides an opinionated Prometheus wrapper with predefined
// metric sets for common patterns (HTTP, gRPC, runtime) and a clean
// functional-options API.
package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Registry wraps a prometheus.Registry with a configured namespace and
// subsystem, providing convenience factories for common metric types.
type Registry struct {
	prometheus *prometheus.Registry
	namespace  string
	subsystem  string
}

// Option configures the Registry.
type Option func(*Registry)

// New creates a Registry with the given options.
func New(opts ...Option) *Registry {
	reg := &Registry{
		prometheus: prometheus.NewRegistry(),
	}

	for _, opt := range opts {
		opt(reg)
	}

	return reg
}

// WithNamespace sets a global namespace prefix for all metrics created through
// this registry (e.g. "myapp").
func WithNamespace(ns string) Option {
	return func(r *Registry) {
		r.namespace = ns
	}
}

// WithSubsystem sets a global subsystem prefix for all metrics created through
// this registry (e.g. "api").
func WithSubsystem(sub string) Option {
	return func(r *Registry) {
		r.subsystem = sub
	}
}

// WithProcessCollector registers OS process metrics (open FDs, virtual memory,
// CPU seconds) on the registry.
func WithProcessCollector() Option {
	return func(r *Registry) {
		r.prometheus.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	}
}

// WithGoCollector registers Go runtime metrics (goroutines, GC stats, memory)
// on the registry.
func WithGoCollector() Option {
	return func(r *Registry) {
		r.prometheus.MustRegister(collectors.NewGoCollector())
	}
}

// PrometheusRegistry returns the underlying *prometheus.Registry so callers
// can integrate with third-party libraries that require it.
func (r *Registry) PrometheusRegistry() *prometheus.Registry {
	return r.prometheus
}

// NewCounter creates, registers, and returns a new prometheus.Counter.
//
//nolint:ireturn // prometheus.Counter has no exported concrete type
func (r *Registry) NewCounter(name, help string) prometheus.Counter {
	counter := prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: r.namespace,
		Subsystem: r.subsystem,
		Name:      name,
		Help:      help,
	})
	r.prometheus.MustRegister(counter)

	return counter
}

// NewCounterVec creates, registers, and returns a new *prometheus.CounterVec.
func (r *Registry) NewCounterVec(name, help string, labels []string) *prometheus.CounterVec {
	counterVec := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: r.namespace,
		Subsystem: r.subsystem,
		Name:      name,
		Help:      help,
	}, labels)
	r.prometheus.MustRegister(counterVec)

	return counterVec
}

// NewGauge creates, registers, and returns a new prometheus.Gauge.
//
//nolint:ireturn // prometheus.Gauge has no exported concrete type
func (r *Registry) NewGauge(name, help string) prometheus.Gauge {
	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: r.namespace,
		Subsystem: r.subsystem,
		Name:      name,
		Help:      help,
	})
	r.prometheus.MustRegister(gauge)

	return gauge
}

// NewGaugeVec creates, registers, and returns a new *prometheus.GaugeVec.
func (r *Registry) NewGaugeVec(name, help string, labels []string) *prometheus.GaugeVec {
	gaugeVec := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: r.namespace,
		Subsystem: r.subsystem,
		Name:      name,
		Help:      help,
	}, labels)
	r.prometheus.MustRegister(gaugeVec)

	return gaugeVec
}

// DefaultHistogramBuckets are sensible defaults for request duration histograms.
var DefaultHistogramBuckets = []float64{
	0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10,
}

// NewHistogram creates, registers, and returns a new prometheus.Histogram.
// If buckets is nil, DefaultHistogramBuckets are used.
//
//nolint:ireturn // prometheus.Histogram has no exported concrete type
func (r *Registry) NewHistogram(name, help string, buckets []float64) prometheus.Histogram {
	if buckets == nil {
		buckets = DefaultHistogramBuckets
	}

	histogram := prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: r.namespace,
		Subsystem: r.subsystem,
		Name:      name,
		Help:      help,
		Buckets:   buckets,
	})
	r.prometheus.MustRegister(histogram)

	return histogram
}

// NewHistogramVec creates, registers, and returns a new *prometheus.HistogramVec.
// If buckets is nil, DefaultHistogramBuckets are used.
func (r *Registry) NewHistogramVec(name, help string, labels []string, buckets []float64) *prometheus.HistogramVec {
	if buckets == nil {
		buckets = DefaultHistogramBuckets
	}

	histogramVec := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: r.namespace,
		Subsystem: r.subsystem,
		Name:      name,
		Help:      help,
		Buckets:   buckets,
	}, labels)
	r.prometheus.MustRegister(histogramVec)

	return histogramVec
}

// Handler returns an http.Handler that serves the collected metrics in
// Prometheus exposition format.
func (r *Registry) Handler() http.Handler {
	return promhttp.HandlerFor(r.prometheus, promhttp.HandlerOpts{})
}
