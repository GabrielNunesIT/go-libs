package metrics

import (
	"context"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// timestampedItem wraps a value with the time it was sent into the channel,
// enabling latency measurement on the receive side.
type timestampedItem[T any] struct {
	value        T
	sentAt       time.Time
	hasTimestamp bool
}

// ChannelMonitor wraps a Go channel with Prometheus instrumentation for
// length, capacity, throughput, and end-to-end latency.
type ChannelMonitor[T any] struct {
	channel    chan timestampedItem[T]
	length     prometheus.Gauge
	capacity   prometheus.Gauge
	throughput *prometheus.CounterVec
	latency    prometheus.Histogram

	closeOnce sync.Once
}

// ChannelOption configures a ChannelMonitor.
type ChannelOption func(*channelConfig)

type channelConfig struct {
	buckets []float64
}

// WithChannelBuckets overrides the default histogram buckets for channel
// latency tracking.
func WithChannelBuckets(buckets []float64) ChannelOption {
	return func(cfg *channelConfig) {
		cfg.buckets = buckets
	}
}

// channelLatencyBuckets are sensible defaults for channel latency,
// skewed toward sub-millisecond ranges since in-process channels are fast.
var channelLatencyBuckets = []float64{
	0.0001, 0.0005, 0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 5,
}

// NewChannelMonitor creates a monitored channel with the given buffer size and
// registers four metrics on the Registry:
//
//   - <name>_length          (gauge)   — current number of items in the channel
//   - <name>_capacity        (gauge)   — channel buffer capacity (constant)
//   - <name>_throughput_total (counter vec: operation=send|receive)
//   - <name>_latency_seconds  (histogram) — time an item spends in the channel
//
// It is a drop-in replacement for a plain Go channel when you need
// observability. Instead of creating a channel the usual way:
//
//	ch := make(chan Job, 100)
//	ch <- job       // send
//	job := <-ch     // receive
//
// Create a ChannelMonitor and use its Send/Receive methods:
//
//	ch := metrics.NewChannelMonitor[Job](reg, "job_queue", 100)
//	ch.Send(ctx, job)          // blocking send with context support
//	job, err := ch.Receive(ctx) // blocking receive with context support
//
// Non-blocking variants are also available via TrySend and TryReceive.
// The monitor automatically tracks length, capacity, throughput, and
// the time each item spends in the channel (latency).
func NewChannelMonitor[T any](reg *Registry, name string, size int, opts ...ChannelOption) *ChannelMonitor[T] {
	cfg := &channelConfig{
		buckets: channelLatencyBuckets,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	monitor := &ChannelMonitor[T]{
		channel: make(chan timestampedItem[T], size),
		length: reg.NewGauge(
			name+"_length",
			"Current number of items in the channel.",
		),
		capacity: reg.NewGauge(
			name+"_capacity",
			"Buffer capacity of the channel.",
		),
		throughput: reg.NewCounterVec(
			name+"_throughput_total",
			"Total items sent to or received from the channel.",
			[]string{"operation"},
		),
		latency: reg.NewHistogram(
			name+"_latency_seconds",
			"Time an item spends in the channel from send to receive.",
			cfg.buckets,
		),
	}

	monitor.capacity.Set(float64(size))

	return monitor
}

// Send sends a value into the channel, blocking until the send succeeds or the
// context is canceled. Returns the context error on cancellation.
func (cm *ChannelMonitor[T]) Send(ctx context.Context, value T) error {
	item := timestampedItem[T]{
		value:        value,
		sentAt:       time.Now(),
		hasTimestamp: true,
	}

	select {
	case cm.channel <- item:
		cm.throughput.WithLabelValues("send").Inc()
		cm.length.Set(float64(len(cm.channel)))

		return nil
	case <-ctx.Done():
		return ctx.Err() //nolint:wrapcheck // propagate context error directly
	}
}

// Receive waits for a value from the channel, blocking until one is available
// or the context is canceled. Returns the context error on cancellation.
//
//nolint:ireturn // generic type parameter T
func (cm *ChannelMonitor[T]) Receive(ctx context.Context) (T, error) {
	select {
	case item := <-cm.channel:
		cm.throughput.WithLabelValues("receive").Inc()
		cm.length.Set(float64(len(cm.channel)))

		if item.hasTimestamp {
			cm.latency.Observe(time.Since(item.sentAt).Seconds())
		}

		return item.value, nil
	case <-ctx.Done():
		var zero T

		return zero, ctx.Err() //nolint:wrapcheck // propagate context error directly
	}
}

// TrySend attempts a non-blocking send. Returns true if the item was sent.
func (cm *ChannelMonitor[T]) TrySend(value T) bool {
	item := timestampedItem[T]{
		value:        value,
		sentAt:       time.Now(),
		hasTimestamp: true,
	}

	select {
	case cm.channel <- item:
		cm.throughput.WithLabelValues("send").Inc()
		cm.length.Set(float64(len(cm.channel)))

		return true
	default:
		return false
	}
}

// TryReceive attempts a non-blocking receive. Returns the value and true if an
// item was available, or the zero value and false otherwise.
//
//nolint:ireturn // generic type parameter T
func (cm *ChannelMonitor[T]) TryReceive() (T, bool) {
	select {
	case item := <-cm.channel:
		cm.throughput.WithLabelValues("receive").Inc()
		cm.length.Set(float64(len(cm.channel)))

		if item.hasTimestamp {
			cm.latency.Observe(time.Since(item.sentAt).Seconds())
		}

		return item.value, true
	default:
		var zero T

		return zero, false
	}
}

// Len returns the current number of items in the channel.
func (cm *ChannelMonitor[T]) Len() int {
	return len(cm.channel)
}

// Cap returns the buffer capacity of the channel.
func (cm *ChannelMonitor[T]) Cap() int {
	return cap(cm.channel)
}

// Close closes the underlying channel. Safe to call multiple times.
func (cm *ChannelMonitor[T]) Close() {
	cm.closeOnce.Do(func() {
		close(cm.channel)
	})
}
