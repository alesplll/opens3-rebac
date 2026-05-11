package env

import (
	"github.com/caarlos0/env/v11"
)

type tracingEnvConfig struct {
	CollectorEndpointValue string `env:"OTEL_EXPORTER_OTLP_ENDPOINT"`
	ServiceNameValue       string `env:"OTEL_SERVICE_NAME"`
	EnvironmentValue       string `env:"OTEL_ENVIRONMENT"`
	ServiceVersionValue    string `env:"OTEL_SERVICE_VERSION"`
}

type tracingConfig struct {
	raw tracingEnvConfig
}

func NewTracingConfig() (*tracingConfig, error) {
	var raw tracingEnvConfig
	err := env.Parse(&raw)
	if err != nil {
		return nil, err
	}

	return &tracingConfig{raw: raw}, nil
}

func (cfg *tracingConfig) CollectorEndpoint() string {
	return cfg.raw.CollectorEndpointValue
}

func (cfg *tracingConfig) ServiceName() string {
	return cfg.raw.ServiceNameValue
}

func (cfg *tracingConfig) Environment() string {
	return cfg.raw.EnvironmentValue
}

func (cfg *tracingConfig) ServiceVersion() string {
	return cfg.raw.ServiceVersionValue
}
