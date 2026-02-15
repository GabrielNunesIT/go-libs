package metrics_test

import (
	"context"
	"testing"
	"time"

	"github.com/GabrielNunesIT/go-libs/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewChannelMonitor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		chanName string
		size     int
	}{
		{
			name:     "buffered channel",
			chanName: "jobs",
			size:     10,
		},
		{
			name:     "unbuffered channel",
			chanName: "signals",
			size:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			reg := metrics.New()
			monitor := metrics.NewChannelMonitor[string](reg, tt.chanName, tt.size)

			assert.NotNil(t, monitor)
			assert.Equal(t, tt.size, monitor.Cap())
			assert.Equal(t, 0, monitor.Len())

			// Verify capacity gauge was set
			families := collectMetricFamilies(t, reg)
			capFam := findFamily(families, tt.chanName+"_capacity")
			require.NotNil(t, capFam)
			assert.InDelta(t, float64(tt.size), capFam.GetMetric()[0].GetGauge().GetValue(), 0.001)
		})
	}
}

func TestChannelMonitorSendReceive(t *testing.T) {
	t.Parallel()

	reg := metrics.New()
	monitor := metrics.NewChannelMonitor[int](reg, "tasks", 5)

	ctx := context.Background()

	// Send 3 items
	for idx := range 3 {
		err := monitor.Send(ctx, idx)
		require.NoError(t, err)
	}

	assert.Equal(t, 3, monitor.Len())

	// Receive all 3
	for expected := range 3 {
		val, err := monitor.Receive(ctx)
		require.NoError(t, err)
		assert.Equal(t, expected, val)
	}

	assert.Equal(t, 0, monitor.Len())

	// Verify throughput counters
	families := collectMetricFamilies(t, reg)
	throughputFam := findFamily(families, "tasks_throughput_total")
	require.NotNil(t, throughputFam)

	for _, metric := range throughputFam.GetMetric() {
		labels := labelPairs(metric)

		switch labels["operation"] {
		case "send":
			assert.InDelta(t, 3.0, metric.GetCounter().GetValue(), 0.001)
		case "receive":
			assert.InDelta(t, 3.0, metric.GetCounter().GetValue(), 0.001)
		}
	}

	// Verify latency histogram was observed
	latencyFam := findFamily(families, "tasks_latency_seconds")
	require.NotNil(t, latencyFam)
	assert.Equal(t, uint64(3), latencyFam.GetMetric()[0].GetHistogram().GetSampleCount())
}

func TestChannelMonitorTrySendTryReceive(t *testing.T) {
	t.Parallel()

	reg := metrics.New()
	monitor := metrics.NewChannelMonitor[string](reg, "events", 2)

	// TrySend should succeed for buffered channel
	assert.True(t, monitor.TrySend("event1"))
	assert.True(t, monitor.TrySend("event2"))

	// TrySend should fail when buffer is full
	assert.False(t, monitor.TrySend("event3"))

	assert.Equal(t, 2, monitor.Len())

	// TryReceive should succeed
	val, ok := monitor.TryReceive()
	assert.True(t, ok)
	assert.Equal(t, "event1", val)

	val, ok = monitor.TryReceive()
	assert.True(t, ok)
	assert.Equal(t, "event2", val)

	// TryReceive on empty channel should return false
	val, ok = monitor.TryReceive()
	assert.False(t, ok)
	assert.Empty(t, val)
}

func TestChannelMonitorContextCancellation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		fn   func(t *testing.T)
	}{
		{
			name: "send cancelled",
			fn: func(t *testing.T) {
				t.Helper()

				reg := metrics.New()
				// Unbuffered channel — Send will block
				monitor := metrics.NewChannelMonitor[int](reg, "blocked_send", 0)

				ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
				defer cancel()

				err := monitor.Send(ctx, 42)
				assert.ErrorIs(t, err, context.DeadlineExceeded)
			},
		},
		{
			name: "receive cancelled",
			fn: func(t *testing.T) {
				t.Helper()

				reg := metrics.New()
				monitor := metrics.NewChannelMonitor[int](reg, "blocked_recv", 5)

				ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
				defer cancel()

				_, err := monitor.Receive(ctx)
				assert.ErrorIs(t, err, context.DeadlineExceeded)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.fn(t)
		})
	}
}

func TestChannelMonitorLatency(t *testing.T) {
	t.Parallel()

	reg := metrics.New()
	monitor := metrics.NewChannelMonitor[string](reg, "latency_test", 1)

	ctx := context.Background()

	err := monitor.Send(ctx, "hello")
	require.NoError(t, err)

	// Small delay to ensure measurable latency
	time.Sleep(5 * time.Millisecond)

	val, err := monitor.Receive(ctx)
	require.NoError(t, err)
	assert.Equal(t, "hello", val)

	families := collectMetricFamilies(t, reg)
	latencyFam := findFamily(families, "latency_test_latency_seconds")
	require.NotNil(t, latencyFam)

	hist := latencyFam.GetMetric()[0].GetHistogram()
	assert.Equal(t, uint64(1), hist.GetSampleCount())
	// Latency should be at least 5ms
	assert.GreaterOrEqual(t, hist.GetSampleSum(), 0.005)
}

func TestChannelMonitorClose(t *testing.T) {
	t.Parallel()

	reg := metrics.New()
	monitor := metrics.NewChannelMonitor[int](reg, "closeable", 5)

	// Close should not panic
	monitor.Close()

	// Double close should not panic either
	monitor.Close()
}

func TestChannelMonitorWithCustomBuckets(t *testing.T) {
	t.Parallel()

	customBuckets := []float64{0.001, 0.01, 0.1}
	reg := metrics.New()
	monitor := metrics.NewChannelMonitor[int](reg, "custom_bucket_chan", 1,
		metrics.WithChannelBuckets(customBuckets),
	)

	ctx := context.Background()

	err := monitor.Send(ctx, 1)
	require.NoError(t, err)

	_, err = monitor.Receive(ctx)
	require.NoError(t, err)

	families := collectMetricFamilies(t, reg)
	latencyFam := findFamily(families, "custom_bucket_chan_latency_seconds")
	require.NotNil(t, latencyFam)

	hist := latencyFam.GetMetric()[0].GetHistogram()
	assert.Len(t, hist.GetBucket(), len(customBuckets))
}

func TestChannelMonitorLengthGauge(t *testing.T) {
	t.Parallel()

	reg := metrics.New()
	monitor := metrics.NewChannelMonitor[int](reg, "gauge_check", 10)

	ctx := context.Background()

	// Send 5 items
	for idx := range 5 {
		err := monitor.Send(ctx, idx)
		require.NoError(t, err)
	}

	families := collectMetricFamilies(t, reg)
	lenFam := findFamily(families, "gauge_check_length")
	require.NotNil(t, lenFam)

	// The gauge should reflect the current length after sends
	gaugeVal := lenFam.GetMetric()[0].GetGauge().GetValue()
	assert.InDelta(t, 5.0, gaugeVal, 1.0) // Allow small race tolerance

	// Receive 2
	for range 2 {
		_, err := monitor.Receive(ctx)
		require.NoError(t, err)
	}

	families = collectMetricFamilies(t, reg)
	lenFam = findFamily(families, "gauge_check_length")
	require.NotNil(t, lenFam)

	gaugeVal = lenFam.GetMetric()[0].GetGauge().GetValue()
	assert.InDelta(t, 3.0, gaugeVal, 1.0)
}
