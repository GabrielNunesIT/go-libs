package metrics

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

const unknownService = "unknown"

// GRPCMetrics provides predefined Prometheus metrics for gRPC servers:
// request counter and request duration histogram.
type GRPCMetrics struct {
	requestsTotal   *prometheus.CounterVec
	requestDuration *prometheus.HistogramVec
	buckets         []float64
}

// GRPCOption configures GRPCMetrics.
type GRPCOption func(*GRPCMetrics)

// WithGRPCBuckets overrides the default histogram buckets for gRPC request
// duration tracking.
func WithGRPCBuckets(buckets []float64) GRPCOption {
	return func(m *GRPCMetrics) {
		m.buckets = buckets
	}
}

// NewGRPCMetrics creates and registers a predefined set of gRPC metrics on the
// given Registry. The following metrics are created:
//
//   - grpc_requests_total (counter vec: method, service, code)
//   - grpc_request_duration_seconds (histogram vec: method, service, code)
func NewGRPCMetrics(reg *Registry, opts ...GRPCOption) *GRPCMetrics {
	grpcMetrics := &GRPCMetrics{
		buckets: DefaultHistogramBuckets,
	}

	for _, opt := range opts {
		opt(grpcMetrics)
	}

	labels := []string{"method", "service", "code"}

	grpcMetrics.requestsTotal = reg.NewCounterVec(
		"grpc_requests_total",
		"Total number of gRPC requests processed.",
		labels,
	)
	grpcMetrics.requestDuration = reg.NewHistogramVec(
		"grpc_request_duration_seconds",
		"Duration of gRPC requests in seconds.",
		labels,
		grpcMetrics.buckets,
	)

	return grpcMetrics
}

// RequestsTotal returns the underlying counter vec.
func (m *GRPCMetrics) RequestsTotal() *prometheus.CounterVec {
	return m.requestsTotal
}

// RequestDuration returns the underlying histogram vec.
func (m *GRPCMetrics) RequestDuration() *prometheus.HistogramVec {
	return m.requestDuration
}

// splitMethodName extracts the service and method from a gRPC full method
// string of the form "/package.Service/Method".
func splitMethodName(fullMethod string) (service, method string) {
	// fullMethod is formatted as "/service/method"
	if fullMethod == "" || fullMethod[0] != '/' {
		return unknownService, fullMethod
	}

	// Trim leading slash
	trimmed := fullMethod[1:]
	pos := 0

	for idx := range len(trimmed) {
		if trimmed[idx] == '/' {
			pos = idx

			break
		}
	}

	if pos == 0 {
		return unknownService, trimmed
	}

	return trimmed[:pos], trimmed[pos+1:]
}

// UnaryServerInterceptor returns a grpc.UnaryServerInterceptor that records
// request count and duration for every unary RPC.
func (m *GRPCMetrics) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		service, method := splitMethodName(info.FullMethod)

		start := time.Now()
		resp, err := handler(ctx, req)
		elapsed := time.Since(start).Seconds()

		code := status.Code(err).String()
		m.requestsTotal.WithLabelValues(method, service, code).Inc()
		m.requestDuration.WithLabelValues(method, service, code).Observe(elapsed)

		return resp, err
	}
}

// wrappedStream wraps grpc.ServerStream to intercept calls.
type wrappedStream struct {
	grpc.ServerStream
}

// StreamServerInterceptor returns a grpc.StreamServerInterceptor that records
// request count and duration for every streaming RPC.
func (m *GRPCMetrics) StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(
		srv any,
		stream grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		service, method := splitMethodName(info.FullMethod)

		start := time.Now()
		err := handler(srv, &wrappedStream{stream})
		elapsed := time.Since(start).Seconds()

		code := status.Code(err).String()
		m.requestsTotal.WithLabelValues(method, service, code).Inc()
		m.requestDuration.WithLabelValues(method, service, code).Observe(elapsed)

		return err
	}
}
