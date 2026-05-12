package env

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

type jwtEnvConfig struct {
	AccessSecret        string `env:"ACCESS_TOKEN_SECRET" envDefault:""`
	RefreshSecret       string `env:"REFRESH_TOKEN_SECRET" envDefault:""`
	LegacyAccessSecret  string `env:"JWT_SECRET" envDefault:""`
	LegacyRefreshSecret string `env:"JWT_REFRESH_SECRET" envDefault:""`
}

type jwtConfig struct {
	raw jwtEnvConfig
}

func NewJWTConfig() (*jwtConfig, error) {
	var raw jwtEnvConfig
	if err := env.Parse(&raw); err != nil {
		return nil, err
	}
	if raw.AccessSecret == "" {
		raw.AccessSecret = raw.LegacyAccessSecret
	}
	if raw.RefreshSecret == "" {
		raw.RefreshSecret = raw.LegacyRefreshSecret
	}
	if raw.AccessSecret == "" {
		return nil, fmt.Errorf("ACCESS_TOKEN_SECRET is required")
	}

	return &jwtConfig{raw: raw}, nil
}

func (c *jwtConfig) AccessTokenSecretKey() string {
	return c.raw.AccessSecret
}

func (c *jwtConfig) RefreshTokenSecretKey() string {
	if c.raw.RefreshSecret != "" {
		return c.raw.RefreshSecret
	}

	return c.raw.AccessSecret
}
