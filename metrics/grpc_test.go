package metrics_test

import (
	"context"
	"errors"
	"testing"

	"github.com/GabrielNunesIT/go-libs/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

func TestNewGRPCMetrics(t *testing.T) {
	t.Parallel()

	reg := metrics.New(metrics.WithNamespace("svc"))
	m := metrics.NewGRPCMetrics(reg)

	assert.NotNil(t, m.RequestsTotal())
	assert.NotNil(t, m.RequestDuration())
}

func TestGRPCUnaryInterceptor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		fullMethod   string
		handlerErr   error
		expectedCode string
		wantService  string
		wantMethod   string
	}{
		{
			name:         "successful call",
			fullMethod:   "/mypackage.MyService/GetUser",
			handlerErr:   nil,
			expectedCode: "OK",
			wantService:  "mypackage.MyService",
			wantMethod:   "GetUser",
		},
		{
			name:         "failed call",
			fullMethod:   "/mypackage.MyService/CreateUser",
			handlerErr:   errors.New("internal error"),
			expectedCode: "Unknown",
			wantService:  "mypackage.MyService",
			wantMethod:   "CreateUser",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			reg := metrics.New()
			m := metrics.NewGRPCMetrics(reg)
			interceptor := m.UnaryServerInterceptor()

			info := &grpc.UnaryServerInfo{FullMethod: tt.fullMethod}
			handler := func(_ context.Context, _ any) (any, error) {
				return "response", tt.handlerErr
			}

			resp, err := interceptor(context.Background(), "request", info, handler)

			if tt.handlerErr != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, "response", resp)
			}

			// Verify metrics
			families, gatherErr := reg.PrometheusRegistry().Gather()
			require.NoError(t, gatherErr)

			counterFam := findFamily(families, "grpc_requests_total")
			require.NotNil(t, counterFam)
			assert.Len(t, counterFam.GetMetric(), 1)
			assert.InDelta(t, 1.0, counterFam.GetMetric()[0].GetCounter().GetValue(), 0.001)

			labelMap := labelPairs(counterFam.GetMetric()[0])
			assert.Equal(t, tt.wantMethod, labelMap["method"])
			assert.Equal(t, tt.wantService, labelMap["service"])
			assert.Equal(t, tt.expectedCode, labelMap["code"])

			histFam := findFamily(families, "grpc_request_duration_seconds")
			require.NotNil(t, histFam)
			assert.Equal(t, uint64(1), histFam.GetMetric()[0].GetHistogram().GetSampleCount())
		})
	}
}

type fakeServerStream struct {
	grpc.ServerStream
	ctx context.Context //nolint:containedctx // test-only mock
}

func (f *fakeServerStream) Context() context.Context {
	return f.ctx
}

func TestGRPCStreamInterceptor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		fullMethod   string
		handlerErr   error
		expectedCode string
	}{
		{
			name:         "successful stream",
			fullMethod:   "/pkg.Svc/StreamData",
			handlerErr:   nil,
			expectedCode: "OK",
		},
		{
			name:         "failed stream",
			fullMethod:   "/pkg.Svc/StreamData",
			handlerErr:   errors.New("stream error"),
			expectedCode: "Unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			reg := metrics.New()
			m := metrics.NewGRPCMetrics(reg)
			interceptor := m.StreamServerInterceptor()

			info := &grpc.StreamServerInfo{FullMethod: tt.fullMethod}
			stream := &fakeServerStream{ctx: context.Background()}

			handler := func(_ any, _ grpc.ServerStream) error {
				return tt.handlerErr
			}

			err := interceptor(nil, stream, info, handler)

			if tt.handlerErr != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			families, gatherErr := reg.PrometheusRegistry().Gather()
			require.NoError(t, gatherErr)

			counterFam := findFamily(families, "grpc_requests_total")
			require.NotNil(t, counterFam)
			assert.InDelta(t, 1.0, counterFam.GetMetric()[0].GetCounter().GetValue(), 0.001)

			labelMap := labelPairs(counterFam.GetMetric()[0])
			assert.Equal(t, tt.expectedCode, labelMap["code"])
		})
	}
}

func TestWithGRPCBuckets(t *testing.T) {
	t.Parallel()

	customBuckets := []float64{0.05, 0.5, 5.0}
	reg := metrics.New()
	m := metrics.NewGRPCMetrics(reg, metrics.WithGRPCBuckets(customBuckets))

	interceptor := m.UnaryServerInterceptor()
	info := &grpc.UnaryServerInfo{FullMethod: "/pkg.S/M"}
	handler := func(_ context.Context, _ any) (any, error) { return nil, nil }

	_, _ = interceptor(context.Background(), nil, info, handler)

	families, err := reg.PrometheusRegistry().Gather()
	require.NoError(t, err)

	histFam := findFamily(families, "grpc_request_duration_seconds")
	require.NotNil(t, histFam)

	hist := histFam.GetMetric()[0].GetHistogram()
	assert.Len(t, hist.GetBucket(), len(customBuckets))
}

func TestGRPCMultipleCalls(t *testing.T) {
	t.Parallel()

	reg := metrics.New()
	m := metrics.NewGRPCMetrics(reg)
	interceptor := m.UnaryServerInterceptor()

	info := &grpc.UnaryServerInfo{FullMethod: "/pkg.Svc/Call"}
	handler := func(_ context.Context, _ any) (any, error) { return nil, nil }

	for range 3 {
		_, _ = interceptor(context.Background(), nil, info, handler)
	}

	families, err := reg.PrometheusRegistry().Gather()
	require.NoError(t, err)

	counterFam := findFamily(families, "grpc_requests_total")
	require.NotNil(t, counterFam)
	assert.InDelta(t, 3.0, counterFam.GetMetric()[0].GetCounter().GetValue(), 0.001)
}
