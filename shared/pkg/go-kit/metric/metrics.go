package metric

import (
	"context"
	"fmt"
	"log"
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/metric"
	noopmetric "go.opentelemetry.io/otel/metric/noop"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	once sync.Once
)

var (
	meter                 metric.Meter
	requestCounter        metric.Int64Counter
	responseCounter       metric.Int64Counter
	histogramResponseTime metric.Float64Histogram
)

// Init инициализирует все инструменты метрик
func Init(_ context.Context, cfg MetricsConfig) error {
	var err error

	requestCounter, err = meter.Int64Counter(
		fmt.Sprintf("grpc_%s_requests_total", cfg.ServiceName()),
	)
	if err != nil {
		return err
	}

	responseCounter, err = meter.Int64Counter(
		fmt.Sprintf("grpc_%s_response_total", cfg.ServiceName()),
	)
	if err != nil {
		return err
	}

	histogramResponseTime, err = meter.Float64Histogram(
		fmt.Sprintf("grpc_%s_histogram_response_time_seconds", cfg.ServiceName()),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(
			0.0001, 0.0002, 0.0004, 0.0008, 0.0016, 0.0032, 0.0064, 0.0128,
			0.0256, 0.0512, 0.1024, 0.2048, 0.4096, 0.8192, 1.6384, 3.2768,
		),
	)
	if err != nil {
		return err
	}

	return nil
}

func IncRequestCounter(ctx context.Context) {
	if requestCounter == nil {
		return
	}

	requestCounter.Add(ctx, 1)
}

func IncResponseCounter(ctx context.Context, status, method string) {
	if responseCounter == nil {
		return
	}

	responseCounter.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("status", status),
			attribute.String("method", method),
		),
	)
}

func HistogramResponseTimeObserve(ctx context.Context, status string, time float64) {
	if histogramResponseTime == nil {
		return
	}

	histogramResponseTime.Record(ctx, time,
		metric.WithAttributes(
			attribute.String("status", status),
		),
	)
}

func Meter() metric.Meter {
	if meter == nil {
		return noopmetric.Meter{}
	}

	return meter
}

func NewInt64Counter(name string, opts ...metric.Int64CounterOption) (metric.Int64Counter, error) {
	return Meter().Int64Counter(name, opts...)
}

func NewInt64ObservableGauge(name string, opts ...metric.Int64ObservableGaugeOption) (metric.Int64ObservableGauge, error) {
	return Meter().Int64ObservableGauge(name, opts...)
}

func RegisterCallback(callback func(context.Context, metric.Observer) error, instruments ...metric.Observable) (metric.Registration, error) {
	return Meter().RegisterCallback(callback, instruments...)
}

func InitOTELMetrics(cfg MetricsConfig) (*sdkmetric.MeterProvider, error) {
	once.Do(func() {
		meter = otel.Meter(cfg.ServiceName())
	})

	exporter, err := otlpmetricgrpc.New(
		context.Background(),
		otlpmetricgrpc.WithEndpoint(cfg.OTLPEndpoint()),
		otlpmetricgrpc.WithTLSCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	res, err := resource.New(
		context.Background(),
		resource.WithAttributes(
			attribute.String("service.name", cfg.ServiceName()),
			attribute.String("service.version", cfg.ServiceVersion()),
			attribute.String("deployment.environment", cfg.ServiceEnvironment()),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(
			sdkmetric.NewPeriodicReader(
				exporter,
				sdkmetric.WithInterval(cfg.PushTimeout()),
			),
		),
	)

	otel.SetMeterProvider(meterProvider)

	log.Println("OpenTelemetry initialized successfully")
	return meterProvider, nil
}
