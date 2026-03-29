package env

import (
	"time"

	"github.com/caarlos0/env/v11"
)

type securityEnvConfig struct {
	MaxLoginAttempts    int           `env:"SECURITY_MAX_LOGIN_ATTEMPTS" envDefault:"5"`
	LoginAttemptsWindow time.Duration `env:"SECURITY_LOGIN_ATTEMPTS_WINDOW" envDefault:"15m"`
}

type securityConfig struct {
	raw securityEnvConfig
}

func NewSecurityConfig() (*securityConfig, error) {
	var raw securityEnvConfig
	if err := env.Parse(&raw); err != nil {
		return nil, err
	}
	return &securityConfig{raw: raw}, nil
}

func (c *securityConfig) MaxLoginAttempts() int8 {
	return int8(c.raw.MaxLoginAttempts)
}

func (c *securityConfig) LoginAttemptsWindow() time.Duration {
	return c.raw.LoginAttemptsWindow
}
