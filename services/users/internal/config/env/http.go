package env

import (
	"net"

	"github.com/caarlos0/env/v11"
)

type httpEnvConfig struct {
	Host string `env:"HTTP_HOST,required"`
	Port string `env:"HTTP_PORT,required"`
}

type httpConfig struct {
	raw httpEnvConfig
}

func NewHTTPConfig() (*httpConfig, error) {
	var raw httpEnvConfig
	if err := env.Parse(&raw); err != nil {
		return nil, err
	}

	return &httpConfig{raw: raw}, nil
}

func (cfg *httpConfig) Address() string {
	return net.JoinHostPort(cfg.raw.Host, cfg.raw.Port)
}
