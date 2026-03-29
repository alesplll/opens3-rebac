package env

import (
	"github.com/caarlos0/env/v11"
)

type loggerEnvConfig struct {
	Level              string `env:"LOGGER_LEVEL,required"`
	AsJson             bool   `env:"LOGGER_AS_JSON,required"`
	EnableOLTP         bool   `env:"LOGGER_ENABLE_OLTP,required"`
	ServiceName        string `env:"OTEL_SERVICE_NAME,required"`
	OTLPEndpoint       string `env:"OTEL_EXPORTER_OTLP_ENDPOINT,required"`
	ServiceEnvironment string `env:"OTEL_ENVIRONMENT,required"`
}

type loggerConfig struct {
	raw loggerEnvConfig
}

func NewLoggerConfig() (*loggerConfig, error) {
	var raw loggerEnvConfig
	if err := env.Parse(&raw); err != nil {
		return nil, err
	}

	return &loggerConfig{raw: raw}, nil
}

func (cfg *loggerConfig) LogLevel() string {
	return cfg.raw.Level
}

func (cfg *loggerConfig) AsJSON() bool {
	return cfg.raw.AsJson
}

func (cfg *loggerConfig) EnableOLTP() bool {
	return cfg.raw.EnableOLTP
}
func (cfg *loggerConfig) ServiceName() string {
	return cfg.raw.ServiceName
}
func (cfg *loggerConfig) ServiceEnvironment() string {
	return cfg.raw.ServiceEnvironment
}
func (cfg *loggerConfig) OTLPEndpoint() string {
	return cfg.raw.OTLPEndpoint
}
