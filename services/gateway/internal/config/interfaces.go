package config

type HTTPConfig interface{ Address() string }
type GRPCClientConfig interface{ Address() string }
type LoggerConfig interface {
	LogLevel() string
	AsJSON() bool
	EnableOLTP() bool
	ServiceName() string
	OTLPEndpoint() string
	ServiceEnvironment() string
}
