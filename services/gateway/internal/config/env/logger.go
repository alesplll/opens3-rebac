package env

import "github.com/caarlos0/env/v11"

type loggerEnvConfig struct {
	Level              string `env:"LOGGER_LEVEL,required"`
	AsJSON             bool   `env:"LOGGER_AS_JSON,required"`
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

func (c *loggerConfig) LogLevel() string {
	return c.raw.Level
}

func (c *loggerConfig) AsJSON() bool {
	return c.raw.AsJSON
}

func (c *loggerConfig) EnableOLTP() bool {
	return c.raw.EnableOLTP
}

func (c *loggerConfig) ServiceName() string {
	return c.raw.ServiceName
}

func (c *loggerConfig) OTLPEndpoint() string {
	return c.raw.OTLPEndpoint
}

func (c *loggerConfig) ServiceEnvironment() string {
	return c.raw.ServiceEnvironment
}
