package env

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
)

type telemetryEnvConfig struct {
	Enabled      bool          `env:"OTEL_ENABLED" envDefault:"false"`
	LogLevel     string        `env:"LOGGER_LEVEL" envDefault:"info"`
	LogAsJSON    bool          `env:"LOGGER_AS_JSON" envDefault:"false"`
	LogOTLP      bool          `env:"LOGGER_ENABLE_OLTP" envDefault:"false"`
	ServiceName  string        `env:"OTEL_SERVICE_NAME" envDefault:"gateway"`
	Version      string        `env:"OTEL_SERVICE_VERSION" envDefault:"dev"`
	Environment  string        `env:"OTEL_ENVIRONMENT" envDefault:"local"`
	Endpoint     string        `env:"OTEL_EXPORTER_OTLP_ENDPOINT" envDefault:"localhost:4317"`
	PushInterval time.Duration `env:"OTEL_METRICS_PUSH_TIMEOUT" envDefault:"30s"`
}

type telemetryConfig struct{ raw telemetryEnvConfig }

func NewTelemetryConfig() (*telemetryConfig, error) {
	var raw telemetryEnvConfig
	if err := env.Parse(&raw); err != nil {
		return nil, err
	}
	if raw.ServiceName == "" || raw.Endpoint == "" || raw.PushInterval <= 0 {
		return nil, fmt.Errorf("invalid telemetry configuration")
	}
	return &telemetryConfig{raw: raw}, nil
}

func (cfg *telemetryConfig) Enabled() bool              { return cfg.raw.Enabled }
func (cfg *telemetryConfig) LogLevel() string           { return cfg.raw.LogLevel }
func (cfg *telemetryConfig) AsJSON() bool               { return cfg.raw.LogAsJSON }
func (cfg *telemetryConfig) EnableOLTP() bool           { return cfg.raw.LogOTLP }
func (cfg *telemetryConfig) ServiceName() string        { return cfg.raw.ServiceName }
func (cfg *telemetryConfig) ServiceVersion() string     { return cfg.raw.Version }
func (cfg *telemetryConfig) ServiceEnvironment() string { return cfg.raw.Environment }
func (cfg *telemetryConfig) Environment() string        { return cfg.raw.Environment }
func (cfg *telemetryConfig) OTLPEndpoint() string       { return cfg.raw.Endpoint }
func (cfg *telemetryConfig) CollectorEndpoint() string  { return cfg.raw.Endpoint }
func (cfg *telemetryConfig) PushTimeout() time.Duration { return cfg.raw.PushInterval }
