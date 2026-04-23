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

func (cfg *metricsConfig) ServiceName() string {
	return cfg.raw.ServiceName
}

func (cfg *metricsConfig) ServiceVersion() string {
	return cfg.raw.ServiceVersion
}

func (cfg *metricsConfig) ServiceEnvironment() string {
	return cfg.raw.ServiceEnvironment
}

func (cfg *metricsConfig) OTLPEndpoint() string {
	return cfg.raw.OTLPEndpoint
}

func (cfg *metricsConfig) PushTimeout() time.Duration {
	return cfg.raw.PushTimeout
}
