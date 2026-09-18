package env

import "github.com/caarlos0/env/v11"

type httpConfig struct{ address string }

func NewHTTPConfig() (*httpConfig, error) {
	var raw struct {
		Address string `env:"GATEWAY_HTTP_ADDR" envDefault:"127.0.0.1:8080"`
	}
	if err := env.Parse(&raw); err != nil {
		return nil, err
	}
	return &httpConfig{address: raw.Address}, nil
}

func (cfg *httpConfig) Address() string { return cfg.address }
