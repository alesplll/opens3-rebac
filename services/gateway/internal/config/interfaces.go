package config

import "time"

type HTTPConfig interface {
	Address() string
	Port() string
	ReadTimeout() time.Duration
	WriteTimeout() time.Duration
	IdleTimeout() time.Duration
	ShutdownTimeout() time.Duration
	MaxUploadSizeBytes() int64
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

type GRPCClientConfig interface {
	Address() string
	Timeout() time.Duration
	StreamTimeout() time.Duration
}

type AuthClientConfig interface {
	GRPCClientConfig
}

type UsersClientConfig interface {
	GRPCClientConfig
}

type JWTConfig interface {
	AccessTokenSecretKey() string
	RefreshTokenSecretKey() string
}
