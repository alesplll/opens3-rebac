package env

import (
	"errors"
	"time"

	"github.com/caarlos0/env/v11"
)

type rateLimiterEnvConfig struct {
	Limit  int64         `env:"RATE_LIMITER_LIMIT" envDefault:"100"`
	Period time.Duration `env:"RATE_LIMITER_PERIOD" envDefault:"1s"`
}

type rateLimiterConfig struct {
	raw rateLimiterEnvConfig
}

func NewRateLimiterConfig() (*rateLimiterConfig, error) {
	var raw rateLimiterEnvConfig
	if err := env.Parse(&raw); err != nil {
		return nil, err
	}

	if raw.Limit < 1 || raw.Period < time.Nanosecond {
		return nil, errors.New("invalid rate limiter settings")
	}

	return &rateLimiterConfig{raw: raw}, nil
}

func (cfg *rateLimiterConfig) Limit() int64 {
	return cfg.raw.Limit
}

func (cfg *rateLimiterConfig) Period() time.Duration {
	return cfg.raw.Period
}
