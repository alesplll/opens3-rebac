package env

import "github.com/caarlos0/env/v11"

type jwtEnvConfig struct {
	AccessSecret  string `env:"JWT_SECRET,required"`
	RefreshSecret string `env:"JWT_REFRESH_SECRET" envDefault:""`
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

func (c *jwtConfig) AccessTokenSecretKey() string {
	return c.raw.AccessSecret
}

func (c *jwtConfig) RefreshTokenSecretKey() string {
	if c.raw.RefreshSecret != "" {
		return c.raw.RefreshSecret
	}

	return c.raw.AccessSecret
}
