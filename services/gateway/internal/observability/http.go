package observability

import (
	"net/http"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

func NewHTTPHandler(next http.Handler) (http.Handler, error) {
	meter := otel.Meter("gateway.http")
	requests, err := meter.Int64Counter("http_server_requests_total")
	if err != nil {
		return nil, err
	}
	duration, err := meter.Float64Histogram("http_server_request_duration_seconds", metric.WithUnit("s"))
	if err != nil {
		return nil, err
	}
	tracer := otel.Tracer("gateway.http")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		route := routeName(r.URL.Path)
		ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))
		ctx, span := tracer.Start(ctx, r.Method+" "+route, trace.WithSpanKind(trace.SpanKindServer))
		defer span.End()

		start := time.Now()
		tracked := &statusWriter{ResponseWriter: w}
		defer func() {
			status := tracked.status
			if status == 0 {
				status = http.StatusOK
			}
			attrs := []attribute.KeyValue{
				attribute.String("http.request.method", r.Method),
				attribute.String("http.route", route),
				attribute.Int("http.response.status_code", status),
			}
			requests.Add(ctx, 1, metric.WithAttributes(attrs...))
			duration.Record(ctx, time.Since(start).Seconds(), metric.WithAttributes(attrs...))
			span.SetAttributes(attrs...)
			if recovered := recover(); recovered != nil {
				span.SetStatus(codes.Error, "response stream aborted")
				panic(recovered)
			}
			if status >= http.StatusInternalServerError {
				span.SetStatus(codes.Error, http.StatusText(status))
			}
		}()

		next.ServeHTTP(tracked, r.WithContext(ctx))
	}), nil
}

func routeName(path string) string {
	switch {
	case path == "/healthz":
		return "/healthz"
	case strings.HasPrefix(path, "/swagger/"):
		return "/swagger/*"
	case strings.Contains(strings.TrimPrefix(path, "/"), "/"):
		return "/{bucket}/{key}"
	default:
		return "unmatched"
	}
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	if w.status != 0 {
		return
	}
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusWriter) Write(data []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(data)
}

func (w *statusWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

func (w *statusWriter) Flush() {
	_ = http.NewResponseController(w.ResponseWriter).Flush()
}
