package tracing

import (
	"context"

	traceidctx "github.com/alesplll/opens3-rebac/shared/pkg/kit/contextx/traceIDctx"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const TraceIDHeader string = "x-trace_id"

func UnaryServerInterceptor(serviceName string) grpc.UnaryServerInterceptor {
	tracer := otel.GetTracerProvider().Tracer(serviceName)
	propagator := otel.GetTextMapPropagator()

	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			md = metadata.New(nil)
		}

		ctx = propagator.Extract(ctx, metadataCarrier(md))

		ctx, span := tracer.Start(
			ctx,
			info.FullMethod,
			trace.WithSpanKind(trace.SpanKindServer),
		)
		defer span.End()

		ctx = AddTraceIDToResponse(ctx)

		resp, err := handler(ctx, req)
		if err != nil {
			span.RecordError(err)
		}

		return resp, err
	}
}

func UnaryClientInterceptor(serviceName string) grpc.UnaryClientInterceptor {
	tracer := otel.GetTracerProvider().Tracer(serviceName)
	propagator := otel.GetTextMapPropagator()

	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		spanName := formatSpanName(ctx, method)

		ctx, span := tracer.Start(
			ctx,
			spanName,
			trace.WithSpanKind(trace.SpanKindClient),
		)
		defer span.End()

		carrier := metadataCarrier(traceidctx.ExtractOutgoingMetadata(ctx))
		propagator.Inject(ctx, carrier)
		ctx = metadata.NewOutgoingContext(ctx, metadata.MD(carrier))

		err := invoker(ctx, method, req, reply, cc, opts...)
		if err != nil {
			trace.SpanFromContext(ctx).RecordError(err)
		}

		return err
	}
}

func formatSpanName(ctx context.Context, method string) string {
	if !trace.SpanContextFromContext(ctx).IsValid() {
		return "client." + method
	}

	return method
}

// AddTraceIDToResponse adds the trace ID to the outgoing gRPC response metadata.
// This enables the client to retrieve the trace ID for subsequent lookup in the tracing system.
func AddTraceIDToResponse(ctx context.Context) context.Context {
	traceID, ok := traceidctx.ExtractTraceIDFromSpan(ctx)
	if !ok {
		return ctx
	}

	md := traceidctx.ExtractOutgoingMetadata(ctx)

	md.Set(TraceIDHeader, traceID)
	return metadata.NewOutgoingContext(ctx, md)
}
