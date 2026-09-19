package env

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

const (
	maxRetrieveChunkSizeBytes = 64 << 20
)

type storageEnvConfig struct {
	DataDir      string `env:"DATA_DIR" envDefault:"/data/blobs"`
	MultipartDir string `env:"MULTIPART_DIR" envDefault:"/data/staging"`
	ChunkSize    int    `env:"STORAGE_RETRIEVE_CHUNK_SIZE_BYTES" envDefault:"1048576"`
}

type storageConfig struct {
	raw storageEnvConfig
}

func NewStorageConfig() (*storageConfig, error) {
	var raw storageEnvConfig
	if err := env.Parse(&raw); err != nil {
		return nil, err
	}
	if raw.ChunkSize < 1 || raw.ChunkSize > maxRetrieveChunkSizeBytes {
		return nil, fmt.Errorf("STORAGE_RETRIEVE_CHUNK_SIZE_BYTES must be between 1 and %d", maxRetrieveChunkSizeBytes)
	}

	return &storageConfig{raw: raw}, nil
}

func (cfg *storageConfig) DataDir() string {
	return cfg.raw.DataDir
}

func (cfg *storageConfig) MultipartDir() string {
	return cfg.raw.MultipartDir
}

func (cfg *storageConfig) ChunkSizeBytes() int {
	return cfg.raw.ChunkSize
}
