package tracing

import "google.golang.org/grpc/metadata"

// metadataCarrier is an adapter between gRPC metadata and OpenTelemetry’s text map format.
// Implements the TextMapCarrier interface for trace context propagation.
type metadataCarrier metadata.MD

func (mc metadataCarrier) Get(key string) string {
	values := metadata.MD(mc).Get(key)
	if len(values) == 0 {
		return ""
	}

	return values[0]
}

func (mc metadataCarrier) Set(key, value string) {
	metadata.MD(mc).Set(key, value)
}

func (mc metadataCarrier) Keys() []string {
	keys := make([]string, 0, len(mc))
	for k := range metadata.MD(mc) {
		keys = append(keys, k)
	}

	return keys
}
