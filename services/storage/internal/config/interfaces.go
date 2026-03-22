package config

import (
	"time"
)

type GRPCConfig interface {
	Address() string
}

type StorageConfig interface {
	DataDir() string
	MultipartDir() string
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
