package config

import (
	"os"
)

type Config struct {
	HTTPAddr         string
	MetadataGRPCAddr string
	StorageGRPCAddr  string
}

func Load() Config {
	return Config{
		HTTPAddr:         valueOrDefault("GATEWAY_HTTP_ADDR", "127.0.0.1:8080"),
		MetadataGRPCAddr: valueOrDefault("METADATA_GRPC_ADDR", "localhost:50052"),
		StorageGRPCAddr:  valueOrDefault("STORAGE_GRPC_ADDR", "localhost:50053"),
	}
}

func valueOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
