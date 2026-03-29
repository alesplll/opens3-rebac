package env

import (
	"net"
	"time"

	"github.com/caarlos0/env/v11"
)

type redisEnvConfig struct {
	Host         string        `env:"CACHE_HOST,notEmpty"`
	InternalPort string        `env:"INTERNAL_CACHE_PORT,notEmpty"`
	ExternalPort string        `env:"EXTERNAL_CACHE_PORT,notEmpty"`
	MaxIdle      int           `env:"CACHE_MAX_IDLE" envDefault:"10"`
	ConnTimeout  time.Duration `env:"CACHE_CONNECTION_TIMEOUT" envDefault:"5s"`
	IdleTimeout  time.Duration `env:"CACHE_IDLE_TIMEOUT" envDefault:"240s"`
}

type redisConfig struct {
	raw redisEnvConfig
}

func NewRedisConfig() (*redisConfig, error) {
	var raw redisEnvConfig
	if err := env.Parse(&raw); err != nil {
		return nil, err
	}
	return &redisConfig{raw: raw}, nil
}

func (c *redisConfig) ExternalAddress() string {
	return net.JoinHostPort(c.raw.Host, c.raw.ExternalPort)
}

func (c *redisConfig) InternalAddress() string {
	return net.JoinHostPort(c.raw.Host, c.raw.InternalPort)
}

func (c *redisConfig) MaxIdle() int8 {
	return int8(c.raw.MaxIdle)
}

func (c *redisConfig) ConnTimeout() time.Duration {
	return c.raw.ConnTimeout
}

func (c *redisConfig) IdleTimeout() time.Duration {
	return c.raw.IdleTimeout
}
