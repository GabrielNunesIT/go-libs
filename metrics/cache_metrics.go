package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

// Cache defines the minimal interface that a cache implementation must satisfy
// to be wrapped by InstrumentedCache. The go-libs cache.Cache[K,V] naturally
// implements this interface.
type Cache[K comparable, V any] interface {
	Get(key K) (V, bool)
	Set(key K, value V)
	Delete(key K)
	Len() int
	Clear()
}

// CacheMetrics holds the Prometheus metrics for a cache. It is embedded inside
// InstrumentedCache but can also be used standalone for manual instrumentation.
type CacheMetrics struct {
	hits      prometheus.Counter
	misses    prometheus.Counter
	sets      prometheus.Counter
	deletes   prometheus.Counter
	evictions prometheus.Counter
	size      prometheus.Gauge
	latency   prometheus.Histogram
}

// CacheOption configures cache metrics.
type CacheOption func(*cacheConfig)

type cacheConfig struct {
	buckets []float64
}

// cacheLatencyBuckets are sensible defaults for cache operation latency,
// skewed toward sub-millisecond ranges since cache lookups are typically fast.
var cacheLatencyBuckets = []float64{
	0.00001, 0.00005, 0.0001, 0.0005, 0.001, 0.005, 0.01, 0.05, 0.1, 0.5,
}

// WithCacheBuckets overrides the default histogram buckets for cache
// operation latency tracking.
func WithCacheBuckets(buckets []float64) CacheOption {
	return func(cfg *cacheConfig) {
		cfg.buckets = buckets
	}
}

// newCacheMetrics creates and registers cache metrics on the given Registry.
func newCacheMetrics(reg *Registry, name string, cfg *cacheConfig) *CacheMetrics {
	return &CacheMetrics{
		hits:    reg.NewCounter(name+"_hits_total", "Total number of cache hits."),
		misses:  reg.NewCounter(name+"_misses_total", "Total number of cache misses."),
		sets:    reg.NewCounter(name+"_sets_total", "Total number of cache set operations."),
		deletes: reg.NewCounter(name+"_deletes_total", "Total number of cache delete operations."),
		evictions: reg.NewCounter(
			name+"_evictions_total",
			"Total number of cache evictions.",
		),
		size: reg.NewGauge(name+"_size", "Current number of items in the cache."),
		latency: reg.NewHistogram(
			name+"_operation_duration_seconds",
			"Duration of cache operations in seconds.",
			cfg.buckets,
		),
	}
}

// RecordEviction records a cache eviction event. Use this when your cache
// evicts an entry (e.g. via an OnEvict callback) since evictions happen
// internally and cannot be auto-detected by the wrapper.
func (cm *CacheMetrics) RecordEviction() {
	cm.evictions.Inc()
}

// SetSize sets the current number of items in the cache.
func (cm *CacheMetrics) SetSize(size float64) {
	cm.size.Set(size)
}

// HitRatio computes the current hit ratio as hits / (hits + misses).
// Returns 0 if no lookups have been recorded. For dashboards prefer
// rate-based PromQL expressions.
func (cm *CacheMetrics) HitRatio() float64 {
	hits := readCounter(cm.hits)
	misses := readCounter(cm.misses)
	total := hits + misses

	if total == 0 {
		return 0
	}

	return hits / total
}

// readCounter extracts the current value from a prometheus.Counter.
func readCounter(counter prometheus.Counter) float64 {
	var metric prometheus.Metric = counter
	dtoMetric := &dto.Metric{}

	if err := metric.Write(dtoMetric); err != nil {
		return 0
	}

	return dtoMetric.GetCounter().GetValue()
}

// InstrumentedCache wraps a Cache implementation with automatic Prometheus
// instrumentation. All Get/Set/Delete/Clear calls are transparently measured.
//
// Usage with the go-libs cache:
//
//	// 1. Create your cache as usual.
//	c := cache.New[string, User](
//	    cache.WithCapacity[string, User](1000),
//	    cache.WithPolicy[string, User](cache.PolicyLRU),
//	)
//
//	// 2. Wrap it with metrics — this is the only change.
//	reg := metrics.New(metrics.WithNamespace("myapp"))
//	ic := metrics.NewInstrumentedCache[string, User](reg, "users", c)
//
//	// 3. Use ic instead of c. Same API.
//	ic.Set("alice", alice)
//	user, ok := ic.Get("alice") // automatically records hit + latency + updates size
type InstrumentedCache[K comparable, V any] struct {
	inner   Cache[K, V]
	Metrics *CacheMetrics
}

// NewInstrumentedCache wraps an existing Cache with Prometheus instrumentation.
// The name parameter is used as a prefix for all metric names.
//
// Metrics registered:
//
//   - <name>_hits_total                  (counter)   — cache hits
//   - <name>_misses_total                (counter)   — cache misses
//   - <name>_sets_total                  (counter)   — set operations
//   - <name>_deletes_total               (counter)   — delete operations
//   - <name>_evictions_total             (counter)   — evictions (call Metrics.RecordEviction())
//   - <name>_size                        (gauge)     — current item count
//   - <name>_operation_duration_seconds  (histogram) — operation latency
func NewInstrumentedCache[K comparable, V any](
	reg *Registry,
	name string,
	inner Cache[K, V],
	opts ...CacheOption,
) *InstrumentedCache[K, V] {
	cfg := &cacheConfig{
		buckets: cacheLatencyBuckets,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	cacheMetrics := newCacheMetrics(reg, name, cfg)
	cacheMetrics.size.Set(float64(inner.Len()))

	return &InstrumentedCache[K, V]{
		inner:   inner,
		Metrics: cacheMetrics,
	}
}

// Get retrieves a value from the cache, automatically recording a hit or miss
// and observing the operation latency.
//
//nolint:ireturn // generic type parameter V
func (ic *InstrumentedCache[K, V]) Get(key K) (V, bool) {
	start := time.Now()
	value, found := ic.inner.Get(key)
	elapsed := time.Since(start).Seconds()

	ic.Metrics.latency.Observe(elapsed)

	if found {
		ic.Metrics.hits.Inc()
	} else {
		ic.Metrics.misses.Inc()
	}

	return value, found
}

// Set adds a value to the cache, automatically recording a set operation,
// observing latency, and updating the size gauge.
func (ic *InstrumentedCache[K, V]) Set(key K, value V) {
	start := time.Now()
	ic.inner.Set(key, value)
	elapsed := time.Since(start).Seconds()

	ic.Metrics.sets.Inc()
	ic.Metrics.latency.Observe(elapsed)
	ic.Metrics.size.Set(float64(ic.inner.Len()))
}

// Delete removes a key from the cache, automatically recording a delete
// operation and updating the size gauge.
func (ic *InstrumentedCache[K, V]) Delete(key K) {
	ic.inner.Delete(key)
	ic.Metrics.deletes.Inc()
	ic.Metrics.size.Set(float64(ic.inner.Len()))
}

// Len returns the current number of items in the cache.
func (ic *InstrumentedCache[K, V]) Len() int {
	return ic.inner.Len()
}

// Clear removes all items from the cache and resets the size gauge to 0.
func (ic *InstrumentedCache[K, V]) Clear() {
	ic.inner.Clear()
	ic.Metrics.size.Set(0)
}
