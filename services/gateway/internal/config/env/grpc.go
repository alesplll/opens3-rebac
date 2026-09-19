package env

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

const maxRetrieveChunkSizeBytes = 64 << 20

type grpcClientConfig struct{ address string }

type storageClientConfig struct {
	address                string
	retrieveChunkSizeBytes int
}

func NewMetadataConfig() (*grpcClientConfig, error) {
	var raw struct {
		Address string `env:"METADATA_GRPC_ADDR" envDefault:"localhost:50052"`
	}
	if err := env.Parse(&raw); err != nil {
		return nil, err
	}
	return &grpcClientConfig{address: raw.Address}, nil
}

func NewStorageConfig() (*storageClientConfig, error) {
	var raw struct {
		Address                string `env:"STORAGE_GRPC_ADDR" envDefault:"localhost:50053"`
		RetrieveChunkSizeBytes int    `env:"STORAGE_RETRIEVE_CHUNK_SIZE_BYTES" envDefault:"1048576"`
	}
	if err := env.Parse(&raw); err != nil {
		return nil, err
	}
	if raw.RetrieveChunkSizeBytes < 1 || raw.RetrieveChunkSizeBytes > maxRetrieveChunkSizeBytes {
		return nil, fmt.Errorf("STORAGE_RETRIEVE_CHUNK_SIZE_BYTES must be between 1 and %d", maxRetrieveChunkSizeBytes)
	}
	return &storageClientConfig{address: raw.Address, retrieveChunkSizeBytes: raw.RetrieveChunkSizeBytes}, nil
}

func (cfg *grpcClientConfig) Address() string    { return cfg.address }
func (cfg *storageClientConfig) Address() string { return cfg.address }
func (cfg *storageClientConfig) RetrieveChunkSizeBytes() int {
	return cfg.retrieveChunkSizeBytes
}
