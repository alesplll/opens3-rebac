package env

import (
	"net"

	"github.com/caarlos0/env/v11"
)

type userGRPCEnvConfig struct {
	Host string `env:"USER_SERVER_GRPC_HOST,notEmpty"`
	Port string `env:"USER_SERVER_GRPC_PORT,notEmpty"`
}

type userGRPCConfig struct {
	raw userGRPCEnvConfig
}

func NewUserGRPCConfig() (*userGRPCConfig, error) {
	var raw userGRPCEnvConfig
	if err := env.Parse(&raw); err != nil {
		return nil, err
	}
	return &userGRPCConfig{raw: raw}, nil
}

func (c *userGRPCConfig) Address() string {
	return net.JoinHostPort(c.raw.Host, c.raw.Port)
}
