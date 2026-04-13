package env

import (
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/caarlos0/env/v11"
)

type grpcClientsEnvConfig struct {
	AuthAddr             string        `env:"AUTH_GRPC_ADDR" envDefault:"auth:50051"`
	AuthZAddr            string        `env:"AUTHZ_GRPC_ADDR" envDefault:"authz:50051"`
	UsersAddr            string        `env:"USERS_GRPC_ADDR" envDefault:"users:50051"`
	MetadataAddr         string        `env:"METADATA_GRPC_ADDR" envDefault:"metadata:50052"`
	StorageAddr          string        `env:"STORAGE_GRPC_ADDR" envDefault:"storage:50053"`
	GRPCTimeout          time.Duration `env:"GRPC_TIMEOUT" envDefault:"5s"`
	LegacyGRPCTimeoutMS  string        `env:"GRPC_TIMEOUT_MS"`
	StorageStreamTimeout time.Duration `env:"STORAGE_STREAM_TIMEOUT" envDefault:"30m"`
}

type authClientConfig struct {
	raw grpcClientsEnvConfig
}

type authzClientConfig struct {
	raw grpcClientsEnvConfig
}

type usersClientConfig struct {
	raw grpcClientsEnvConfig
}

type metadataClientConfig struct {
	raw grpcClientsEnvConfig
}

type storageClientConfig struct {
	raw grpcClientsEnvConfig
}

func NewAuthClientConfig() (*authClientConfig, error) {
	raw, err := newGRPCClientsEnvConfig()
	if err != nil {
		return nil, err
	}

	return &authClientConfig{raw: raw}, nil
}

func NewAuthZClientConfig() (*authzClientConfig, error) {
	raw, err := newGRPCClientsEnvConfig()
	if err != nil {
		return nil, err
	}

	return &authzClientConfig{raw: raw}, nil
}

func NewUsersClientConfig() (*usersClientConfig, error) {
	raw, err := newGRPCClientsEnvConfig()
	if err != nil {
		return nil, err
	}

	return &usersClientConfig{raw: raw}, nil
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

	if raw.LegacyGRPCTimeoutMS != "" {
		timeoutMS, err := strconv.Atoi(raw.LegacyGRPCTimeoutMS)
		if err != nil {
			return grpcClientsEnvConfig{}, fmt.Errorf("parse GRPC_TIMEOUT_MS as milliseconds: %w", err)
		}
		raw.GRPCTimeout = time.Duration(timeoutMS) * time.Millisecond
	}

	return raw, nil
}

func (c *authClientConfig) Address() string {
	return normalizeAddress(c.raw.AuthAddr)
}

func (c *authClientConfig) Timeout() time.Duration {
	return c.raw.GRPCTimeout
}

func (c *authClientConfig) StreamTimeout() time.Duration {
	return c.raw.GRPCTimeout
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

func (c *usersClientConfig) Address() string {
	return normalizeAddress(c.raw.UsersAddr)
}

func (c *usersClientConfig) Timeout() time.Duration {
	return c.raw.GRPCTimeout
}

func (c *usersClientConfig) StreamTimeout() time.Duration {
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
