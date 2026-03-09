package traceidctx

import (
	"context"

	"github.com/alesplll/opens3-rebac/shared/pkg/kit/contextx"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/metadata"
)

const TraceIDKey contextx.CtxKey = "trace_id"

func InjectTraceId(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, TraceIDKey, traceID)
}

func ExtractTraceId(ctx context.Context) (string, bool) {
	if traceID, ok := ctx.Value(TraceIDKey).(string); ok {
		return traceID, true
	}
	return "", false
}

func ExtractTraceIDFromSpan(ctx context.Context) (string, bool) {
	span, ok := ExtractSpan(ctx)
	if !ok {
		return "", false
	}
	return span.SpanContext().SpanID().String(), true
}

func ExtractSpan(ctx context.Context) (trace.Span, bool) {
	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		return nil, false
	}

	return span, true
}

func ExtractOutgoingMetadata(ctx context.Context) metadata.MD {
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		return metadata.New(nil)
	}

	return md.Copy()
}
