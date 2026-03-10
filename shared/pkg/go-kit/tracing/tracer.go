package tracing

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.30.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	DefaultCompressor           = "gzip"
	DefaultRetryEnabled         = true
	DefaultRetryInitialInterval = 500 * time.Millisecond
	DefaultRetryMaxInterval     = 5 * time.Second
	DefaultRetryMaxElapsedTime  = 30 * time.Second
	DefaultTimeout              = 5 * time.Second
)

var serviceName string

type Config interface {
	CollectorEndpoint() string
	ServiceName() string
	Environment() string
	ServiceVersion() string
}

// InitTracer initializes the global OpenTelemetry tracer.
// The function returns an error if the initialization fails.
func InitTracer(ctx context.Context, cfg Config) error {
	serviceName = cfg.ServiceName()

	// Create an exporter to send traces to the OpenTelemetry Collector via gRPC
	exporter, err := otlptracegrpc.New(
		ctx,
		otlptracegrpc.WithEndpoint(cfg.CollectorEndpoint()), // Collector address
		otlptracegrpc.WithInsecure(),                        // Disable TLS for local development
		otlptracegrpc.WithTimeout(DefaultTimeout),
		otlptracegrpc.WithCompressor(DefaultCompressor),
		otlptracegrpc.WithRetry(otlptracegrpc.RetryConfig{
			Enabled:         DefaultRetryEnabled,
			InitialInterval: DefaultRetryInitialInterval,
			MaxInterval:     DefaultRetryMaxInterval,
			MaxElapsedTime:  DefaultRetryMaxElapsedTime,
		}),
	)
	if err != nil {
		return err
	}

	// Create a resource with service metadata
	// The resource adds attributes to each trace to help identify the source
	attributeResource, err := resource.New(ctx,
		resource.WithAttributes(
			// Use standard OpenTelemetry attributes
			semconv.ServiceName(cfg.ServiceName()),
			semconv.ServiceVersion(cfg.ServiceVersion()),
			attribute.String("environment", cfg.Environment()),
		),
		// Automatically detect host, OS, and other system attributes
		resource.WithHost(),
		resource.WithOS(),
		resource.WithProcess(),
		resource.WithContainer(),
		resource.WithTelemetrySDK(),
	)
	if err != nil {
		return err
	}

	// Create a tracer provider with the configured exporter and resource
	// BatchSpanProcessor batches spans for efficient export
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(attributeResource),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(1.0))),
	)
	otel.SetTracerProvider(tracerProvider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return nil
}

func ShutdownTracer(ctx context.Context) error {
	provider := otel.GetTracerProvider()
	if provider == nil {
		return nil
	}

	tracerProvider, ok := provider.(*sdktrace.TracerProvider)
	if !ok {
		return nil
	}

	err := tracerProvider.Shutdown(ctx)
	if err != nil {
		return err
	}

	return nil
}

func StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return otel.Tracer(serviceName).Start(ctx, name, opts...)
}

func SpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

func TraceIDFromContext(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		return ""
	}

	return span.SpanContext().TraceID().String()
}
