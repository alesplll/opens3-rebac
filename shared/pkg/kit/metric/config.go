package metric

import "time"

type MetricsConfig interface {
	ServiceName() string
	ServiceVersion() string
	OTLPEndpoint() string
	ServiceEnvironment() string
	PushTimeout() time.Duration
}
