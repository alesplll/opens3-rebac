package env

import "github.com/caarlos0/env/v11"

type grpcClientConfig struct{ address string }

func NewMetadataConfig() (*grpcClientConfig, error) {
	var raw struct {
		Address string `env:"METADATA_GRPC_ADDR" envDefault:"localhost:50052"`
	}
	if err := env.Parse(&raw); err != nil {
		return nil, err
	}
	return &grpcClientConfig{address: raw.Address}, nil
}

func NewStorageConfig() (*grpcClientConfig, error) {
	var raw struct {
		Address string `env:"STORAGE_GRPC_ADDR" envDefault:"localhost:50053"`
	}
	if err := env.Parse(&raw); err != nil {
		return nil, err
	}
	return &grpcClientConfig{address: raw.Address}, nil
}

func (cfg *grpcClientConfig) Address() string { return cfg.address }
