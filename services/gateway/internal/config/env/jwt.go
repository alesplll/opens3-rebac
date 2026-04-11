package env

import "github.com/caarlos0/env/v11"

type jwtEnvConfig struct {
	AccessSecret string `env:"JWT_SECRET,required"`
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
