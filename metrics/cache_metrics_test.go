package metrics_test

import (
	"testing"

	"github.com/GabrielNunesIT/go-libs/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeCache is a minimal in-memory cache for testing InstrumentedCache.
type fakeCache[K comparable, V any] struct {
	items map[K]V
}

func newFakeCache[K comparable, V any]() *fakeCache[K, V] {
	return &fakeCache[K, V]{items: make(map[K]V)}
}

func (fc *fakeCache[K, V]) Get(key K) (V, bool) {
	val, ok := fc.items[key]

	return val, ok
}

func (fc *fakeCache[K, V]) Set(key K, value V) {
	fc.items[key] = value
}

func (fc *fakeCache[K, V]) Delete(key K) {
	delete(fc.items, key)
}

func (fc *fakeCache[K, V]) Len() int {
	return len(fc.items)
}

func (fc *fakeCache[K, V]) Clear() {
	fc.items = make(map[K]V)
}

func TestNewInstrumentedCache(t *testing.T) {
	t.Parallel()

	reg := metrics.New(metrics.WithNamespace("app"))
	inner := newFakeCache[string, int]()
	ic := metrics.NewInstrumentedCache[string, int](reg, "sessions", inner)

	assert.NotNil(t, ic)
	assert.NotNil(t, ic.Metrics)

	families := collectMetricFamilies(t, reg)
	assert.NotNil(t, findFamily(families, "app_sessions_hits_total"))
	assert.NotNil(t, findFamily(families, "app_sessions_misses_total"))
	assert.NotNil(t, findFamily(families, "app_sessions_sets_total"))
	assert.NotNil(t, findFamily(families, "app_sessions_deletes_total"))
	assert.NotNil(t, findFamily(families, "app_sessions_evictions_total"))
	assert.NotNil(t, findFamily(families, "app_sessions_size"))
	assert.NotNil(t, findFamily(families, "app_sessions_operation_duration_seconds"))
}

func TestInstrumentedCacheGetHitMiss(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		seed          map[string]int
		lookups       []string
		expectedHits  float64
		expectedMiss  float64
		expectedRatio float64
	}{
		{
			name:          "all hits",
			seed:          map[string]int{"a": 1, "b": 2},
			lookups:       []string{"a", "b", "a"},
			expectedHits:  3,
			expectedMiss:  0,
			expectedRatio: 1.0,
		},
		{
			name:          "all misses",
			seed:          map[string]int{},
			lookups:       []string{"x", "y"},
			expectedHits:  0,
			expectedMiss:  2,
			expectedRatio: 0.0,
		},
		{
			name:          "mixed",
			seed:          map[string]int{"a": 1},
			lookups:       []string{"a", "b"},
			expectedHits:  1,
			expectedMiss:  1,
			expectedRatio: 0.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			reg := metrics.New()
			inner := newFakeCache[string, int]()

			for key, val := range tt.seed {
				inner.Set(key, val)
			}

			ic := metrics.NewInstrumentedCache[string, int](reg, "c", inner)

			for _, key := range tt.lookups {
				ic.Get(key)
			}

			assert.InDelta(t, tt.expectedRatio, ic.Metrics.HitRatio(), 0.001)

			families := collectMetricFamilies(t, reg)

			if tt.expectedHits > 0 {
				hitsFam := findFamily(families, "c_hits_total")
				require.NotNil(t, hitsFam)
				assert.InDelta(t, tt.expectedHits, hitsFam.GetMetric()[0].GetCounter().GetValue(), 0.001)
			}

			if tt.expectedMiss > 0 {
				missFam := findFamily(families, "c_misses_total")
				require.NotNil(t, missFam)
				assert.InDelta(t, tt.expectedMiss, missFam.GetMetric()[0].GetCounter().GetValue(), 0.001)
			}

			// Verify latency histogram counted all lookups
			histFam := findFamily(families, "c_operation_duration_seconds")
			require.NotNil(t, histFam)
			assert.Equal(t,
				uint64(len(tt.lookups)),
				histFam.GetMetric()[0].GetHistogram().GetSampleCount(),
			)
		})
	}
}

func TestInstrumentedCacheSetUpdatesSize(t *testing.T) {
	t.Parallel()

	reg := metrics.New()
	inner := newFakeCache[string, string]()
	ic := metrics.NewInstrumentedCache[string, string](reg, "store", inner)

	ic.Set("a", "1")
	ic.Set("b", "2")
	ic.Set("c", "3")

	assert.Equal(t, 3, ic.Len())

	families := collectMetricFamilies(t, reg)

	setsFam := findFamily(families, "store_sets_total")
	require.NotNil(t, setsFam)
	assert.InDelta(t, 3.0, setsFam.GetMetric()[0].GetCounter().GetValue(), 0.001)

	sizeFam := findFamily(families, "store_size")
	require.NotNil(t, sizeFam)
	assert.InDelta(t, 3.0, sizeFam.GetMetric()[0].GetGauge().GetValue(), 0.001)
}

func TestInstrumentedCacheDelete(t *testing.T) {
	t.Parallel()

	reg := metrics.New()
	inner := newFakeCache[string, int]()
	inner.Set("a", 1)
	inner.Set("b", 2)

	ic := metrics.NewInstrumentedCache[string, int](reg, "del", inner)

	ic.Delete("a")

	assert.Equal(t, 1, ic.Len())

	families := collectMetricFamilies(t, reg)

	delFam := findFamily(families, "del_deletes_total")
	require.NotNil(t, delFam)
	assert.InDelta(t, 1.0, delFam.GetMetric()[0].GetCounter().GetValue(), 0.001)

	sizeFam := findFamily(families, "del_size")
	require.NotNil(t, sizeFam)
	assert.InDelta(t, 1.0, sizeFam.GetMetric()[0].GetGauge().GetValue(), 0.001)
}

func TestInstrumentedCacheClear(t *testing.T) {
	t.Parallel()

	reg := metrics.New()
	inner := newFakeCache[string, int]()
	inner.Set("a", 1)
	inner.Set("b", 2)

	ic := metrics.NewInstrumentedCache[string, int](reg, "clr", inner)
	ic.Clear()

	assert.Equal(t, 0, ic.Len())

	families := collectMetricFamilies(t, reg)
	sizeFam := findFamily(families, "clr_size")
	require.NotNil(t, sizeFam)
	assert.InDelta(t, 0.0, sizeFam.GetMetric()[0].GetGauge().GetValue(), 0.001)
}

func TestInstrumentedCacheEvictionManual(t *testing.T) {
	t.Parallel()

	reg := metrics.New()
	inner := newFakeCache[string, int]()
	ic := metrics.NewInstrumentedCache[string, int](reg, "evict", inner)

	// Eviction tracking is manual since evictions happen inside the cache
	ic.Metrics.RecordEviction()
	ic.Metrics.RecordEviction()
	ic.Metrics.RecordEviction()

	families := collectMetricFamilies(t, reg)
	evictFam := findFamily(families, "evict_evictions_total")
	require.NotNil(t, evictFam)
	assert.InDelta(t, 3.0, evictFam.GetMetric()[0].GetCounter().GetValue(), 0.001)
}

func TestInstrumentedCacheWithCustomBuckets(t *testing.T) {
	t.Parallel()

	customBuckets := []float64{0.001, 0.01, 0.1}
	reg := metrics.New()
	inner := newFakeCache[string, int]()
	ic := metrics.NewInstrumentedCache[string, int](reg, "custom", inner,
		metrics.WithCacheBuckets(customBuckets),
	)

	ic.Set("a", 1)
	ic.Get("a")

	families := collectMetricFamilies(t, reg)
	histFam := findFamily(families, "custom_operation_duration_seconds")
	require.NotNil(t, histFam)

	hist := histFam.GetMetric()[0].GetHistogram()
	assert.Len(t, hist.GetBucket(), len(customBuckets))
}

func TestInstrumentedCacheHitRatioNoLookups(t *testing.T) {
	t.Parallel()

	reg := metrics.New()
	inner := newFakeCache[string, int]()
	ic := metrics.NewInstrumentedCache[string, int](reg, "empty", inner)

	assert.InDelta(t, 0.0, ic.Metrics.HitRatio(), 0.001)
}
