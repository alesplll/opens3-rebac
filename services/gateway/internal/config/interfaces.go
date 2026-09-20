package config

import "time"

type HTTPConfig interface{ Address() string }
type GRPCClientConfig interface{ Address() string }
type StorageClientConfig interface {
	GRPCClientConfig
	RetrieveChunkSizeBytes() int
}
type LoggerConfig interface {
	LogLevel() string
	AsJSON() bool
	EnableOLTP() bool
	ServiceName() string
	OTLPEndpoint() string
	ServiceEnvironment() string
}

type TelemetryConfig interface {
	LoggerConfig
	Enabled() bool
	ServiceVersion() string
	Environment() string
	CollectorEndpoint() string
	PushTimeout() time.Duration
}
