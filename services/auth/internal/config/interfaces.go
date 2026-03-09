package config

import (
	"time"
)

type GRPCConfig interface {
	Address() string
}

type LoggerConfig interface {
	LogLevel() string
	AsJSON() bool
	EnableOLTP() bool
	ServiceName() string
	OTLPEndpoint() string
	ServiceEnvironment() string
}

type RedisConfig interface {
	ExternalAddress() string
	InternalAddress() string
	MaxIdle() int8
	ConnTimeout() time.Duration
	IdleTimeout() time.Duration
}

type SecurityConfig interface {
	MaxLoginAttempts() int8
	LoginAttemptsWindow() time.Duration
}

type JWTConfig interface {
	RefreshTokenSecretKey() string
	AccessTokenSecretKey() string

	RefreshTokenExpiration() time.Duration
	AccessTokenExpiration() time.Duration
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
