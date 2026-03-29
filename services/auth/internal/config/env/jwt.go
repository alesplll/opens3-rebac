package env

import (
	"time"

	"github.com/caarlos0/env/v11"
)

type jwtEnvConfig struct {
	RefreshSecret string        `env:"REFRESH_TOKEN_SECRET,notEmpty"`
	AccessSecret  string        `env:"ACCESS_TOKEN_SECRET,notEmpty"`
	RefreshTTL    time.Duration `env:"REFRESH_TOKEN_TTL",envDefault:"5m"`
	AccessTTL     time.Duration `env:"ACCESS_TOKEN_TTL",envDefault:"2m`
}

type jwtConfig struct {
	raw jwtEnvConfig
}

func NewJWTConfig() (*jwtConfig, error) {
	var raw jwtEnvConfig
	if err := env.Parse(&raw); err != nil {
		return nil, err
	}
	return &jwtConfig{raw: raw}, nil
}

func (c *jwtConfig) RefreshTokenSecretKey() string {
	return c.raw.RefreshSecret
}

func (c *jwtConfig) AccessTokenSecretKey() string {
	return c.raw.AccessSecret
}

func (c *jwtConfig) RefreshTokenExpiration() time.Duration {
	return c.raw.RefreshTTL
}

func (c *jwtConfig) AccessTokenExpiration() time.Duration {
	return c.raw.AccessTTL
}
