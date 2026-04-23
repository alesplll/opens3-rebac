package config

import "time"

type GRPCConfig interface {
	Address() string
}

type PGConfig interface {
	DSN() string
	Timeout() time.Duration
	NeedLog() bool
}

type KafkaConfig interface {
	BootstrapServers() string
	ObjectDeletedTopic() string
	BucketDeletedTopic() string
}

type LoggerConfig interface {
	LogLevel() string
	AsJSON() bool
	EnableOLTP() bool
	ServiceName() string
	OTLPEndpoint() string
	ServiceEnvironment() string
}

type TracingConfig interface {
	CollectorEndpoint() string
	ServiceName() string
	Environment() string
	ServiceVersion() string
}

type MetricsConfig interface {
	ServiceName() string
	ServiceVersion() string
	OTLPEndpoint() string
	ServiceEnvironment() string
	PushTimeout() time.Duration
}

type RateLimiterConfig interface {
	Limit() int64
	Period() time.Duration
}
