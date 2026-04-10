package env

import (
	"net"
	"time"

	"github.com/caarlos0/env/v11"
)

type grpcClientsEnvConfig struct {
	AuthZAddr            string        `env:"AUTHZ_GRPC_ADDR" envDefault:"authz:50051"`
	MetadataAddr         string        `env:"METADATA_GRPC_ADDR" envDefault:"metadata:50052"`
	StorageAddr          string        `env:"STORAGE_GRPC_ADDR" envDefault:"storage:50053"`
	GRPCTimeout          time.Duration `env:"GRPC_TIMEOUT_MS" envDefault:"5s"`
	StorageStreamTimeout time.Duration `env:"STORAGE_STREAM_TIMEOUT" envDefault:"30m"`
}

type authzClientConfig struct {
	raw grpcClientsEnvConfig
}

type metadataClientConfig struct {
	raw grpcClientsEnvConfig
}

type storageClientConfig struct {
	raw grpcClientsEnvConfig
}

func NewAuthZClientConfig() (*authzClientConfig, error) {
	raw, err := newGRPCClientsEnvConfig()
	if err != nil {
		return nil, err
	}

	return &authzClientConfig{raw: raw}, nil
}

func NewMetadataClientConfig() (*metadataClientConfig, error) {
	raw, err := newGRPCClientsEnvConfig()
	if err != nil {
		return nil, err
	}

	return &metadataClientConfig{raw: raw}, nil
}

func NewStorageClientConfig() (*storageClientConfig, error) {
	raw, err := newGRPCClientsEnvConfig()
	if err != nil {
		return nil, err
	}

	return &storageClientConfig{raw: raw}, nil
}

func newGRPCClientsEnvConfig() (grpcClientsEnvConfig, error) {
	var raw grpcClientsEnvConfig
	if err := env.Parse(&raw); err != nil {
		return grpcClientsEnvConfig{}, err
	}

	return raw, nil
}

func (c *authzClientConfig) Address() string {
	return normalizeAddress(c.raw.AuthZAddr)
}

func (c *authzClientConfig) Timeout() time.Duration {
	return c.raw.GRPCTimeout
}

func (c *authzClientConfig) StreamTimeout() time.Duration {
	return c.raw.GRPCTimeout
}

func (c *metadataClientConfig) Address() string {
	return normalizeAddress(c.raw.MetadataAddr)
}

func (c *metadataClientConfig) Timeout() time.Duration {
	return c.raw.GRPCTimeout
}

func (c *metadataClientConfig) StreamTimeout() time.Duration {
	return c.raw.GRPCTimeout
}

func (c *storageClientConfig) Address() string {
	return normalizeAddress(c.raw.StorageAddr)
}

func (c *storageClientConfig) Timeout() time.Duration {
	return c.raw.GRPCTimeout
}

func (c *storageClientConfig) StreamTimeout() time.Duration {
	return c.raw.StorageStreamTimeout
}

func normalizeAddress(addr string) string {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}

	return net.JoinHostPort(host, port)
}
