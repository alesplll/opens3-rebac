package metrics

import (
	"context"
	"time"

	"github.com/alesplll/opens3-rebac/shared/pkg/kit/metric"
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
