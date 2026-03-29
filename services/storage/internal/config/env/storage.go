package env

import (
	"github.com/caarlos0/env/v11"
)

type storageEnvConfig struct {
	DataDir      string `env:"DATA_DIR" envDefault:"/data/blobs"`
	MultipartDir string `env:"MULTIPART_DIR" envDefault:"/data/multipart"`
}

type storageConfig struct {
	raw storageEnvConfig
}

func NewStorageConfig() (*storageConfig, error) {
	var raw storageEnvConfig
	if err := env.Parse(&raw); err != nil {
		return nil, err
	}

	return &storageConfig{raw: raw}, nil
}

func (cfg *storageConfig) DataDir() string {
	return cfg.raw.DataDir
}

func (cfg *storageConfig) MultipartDir() string {
	return cfg.raw.MultipartDir
}
