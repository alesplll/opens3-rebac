package env

import "github.com/caarlos0/env/v11"

type tracingEnvConfig struct {
	CollectorEndpointValue string `env:"OTEL_EXPORTER_OTLP_ENDPOINT,required"`
	ServiceNameValue       string `env:"OTEL_SERVICE_NAME,required"`
	EnvironmentValue       string `env:"OTEL_ENVIRONMENT,required"`
	ServiceVersionValue    string `env:"OTEL_SERVICE_VERSION,required"`
}

type tracingConfig struct {
	raw tracingEnvConfig
}

func NewTracingConfig() (*tracingConfig, error) {
	var raw tracingEnvConfig
	if err := env.Parse(&raw); err != nil {
		return nil, err
	}

	return &tracingConfig{raw: raw}, nil
}

func (c *tracingConfig) CollectorEndpoint() string {
	return c.raw.CollectorEndpointValue
}

func (c *tracingConfig) ServiceName() string {
	return c.raw.ServiceNameValue
}

func (c *tracingConfig) Environment() string {
	return c.raw.EnvironmentValue
}

func (c *tracingConfig) ServiceVersion() string {
	return c.raw.ServiceVersionValue
}
