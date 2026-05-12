package env

import (
	"time"

	"github.com/caarlos0/env/v11"
)

type metricsEnvConfig struct {
	ServiceName        string        `env:"OTEL_SERVICE_NAME,required"`
	ServiceVersion     string        `env:"OTEL_SERVICE_VERSION,required"`
	ServiceEnvironment string        `env:"OTEL_ENVIRONMENT,required"`
	OTLPEndpoint       string        `env:"OTEL_EXPORTER_OTLP_ENDPOINT,required"`
	PushTimeout        time.Duration `env:"OTEL_METRICS_PUSH_TIMEOUT,required"`
}

type metricsConfig struct {
	raw metricsEnvConfig
}

func NewMetricsConfig() (*metricsConfig, error) {
	var raw metricsEnvConfig
	if err := env.Parse(&raw); err != nil {
		return nil, err
	}

	return &metricsConfig{raw: raw}, nil
}

func (c *metricsConfig) ServiceName() string {
	return c.raw.ServiceName
}

func (c *metricsConfig) ServiceVersion() string {
	return c.raw.ServiceVersion
}

func (c *metricsConfig) OTLPEndpoint() string {
	return c.raw.OTLPEndpoint
}

func (c *metricsConfig) ServiceEnvironment() string {
	return c.raw.ServiceEnvironment
}

func (c *metricsConfig) PushTimeout() time.Duration {
	return c.raw.PushTimeout
}
