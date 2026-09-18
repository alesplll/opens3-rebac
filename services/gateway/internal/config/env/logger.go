package env

import "github.com/caarlos0/env/v11"

type loggerConfig struct {
	level       string
	asJSON      bool
	enableOTLP  bool
	serviceName string
	endpoint    string
	environment string
}

func NewLoggerConfig() (*loggerConfig, error) {
	var raw struct {
		Level       string `env:"LOGGER_LEVEL" envDefault:"info"`
		AsJSON      bool   `env:"LOGGER_AS_JSON" envDefault:"false"`
		EnableOTLP  bool   `env:"LOGGER_ENABLE_OLTP" envDefault:"false"`
		ServiceName string `env:"OTEL_SERVICE_NAME" envDefault:"gateway"`
		Endpoint    string `env:"OTEL_EXPORTER_OTLP_ENDPOINT" envDefault:"localhost:4317"`
		Environment string `env:"OTEL_ENVIRONMENT" envDefault:"development"`
	}
	if err := env.Parse(&raw); err != nil {
		return nil, err
	}
	return &loggerConfig{raw.Level, raw.AsJSON, raw.EnableOTLP, raw.ServiceName, raw.Endpoint, raw.Environment}, nil
}

func (cfg *loggerConfig) LogLevel() string           { return cfg.level }
func (cfg *loggerConfig) AsJSON() bool               { return cfg.asJSON }
func (cfg *loggerConfig) EnableOLTP() bool           { return cfg.enableOTLP }
func (cfg *loggerConfig) ServiceName() string        { return cfg.serviceName }
func (cfg *loggerConfig) OTLPEndpoint() string       { return cfg.endpoint }
func (cfg *loggerConfig) ServiceEnvironment() string { return cfg.environment }
