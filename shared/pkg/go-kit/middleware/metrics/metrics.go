package metrics

import (
	"context"
	"time"

	"github.com/alesplll/opens3-rebac/shared/pkg/go-kit/metric"
	"google.golang.org/grpc"
)

func MetricsInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	metric.IncRequestCounter(ctx)

	timeStart := time.Now()

	res, err := handler(ctx, req)
	diffTime := time.Since(timeStart)

	if err != nil {
		metric.IncResponseCounter(ctx, "error", info.FullMethod)
		metric.HistogramResponseTimeObserve(ctx, "error", diffTime.Seconds())
	} else {
		metric.IncResponseCounter(ctx, "success", info.FullMethod)
		metric.HistogramResponseTimeObserve(ctx, "success", diffTime.Seconds())
	}

	return res, err
}

func StreamMetricsInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	metric.IncRequestCounter(ss.Context())

	timeStart := time.Now()
	err := handler(srv, ss)
	diffTime := time.Since(timeStart)

	if err != nil {
		metric.IncResponseCounter(ss.Context(), "error", info.FullMethod)
		metric.HistogramResponseTimeObserve(ss.Context(), "error", diffTime.Seconds())
	} else {
		metric.IncResponseCounter(ss.Context(), "success", info.FullMethod)
		metric.HistogramResponseTimeObserve(ss.Context(), "success", diffTime.Seconds())
	}

	return err
}
