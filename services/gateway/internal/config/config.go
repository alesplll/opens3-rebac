package config

import (
	"errors"
	"os"

	"github.com/alesplll/opens3-rebac/services/gateway/internal/config/env"
	"github.com/joho/godotenv"
)

type Config struct {
	HTTP     HTTPConfig
	Metadata GRPCClientConfig
	Storage  GRPCClientConfig
	Logger   LoggerConfig
}

// Load uses the process environment and, when present, a local .env file.
func Load(path ...string) (*Config, error) {
	if err := godotenv.Load(path...); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	httpConfig, err := env.NewHTTPConfig()
	if err != nil {
		return nil, err
	}
	metadataConfig, err := env.NewMetadataConfig()
	if err != nil {
		return nil, err
	}
	storageConfig, err := env.NewStorageConfig()
	if err != nil {
		return nil, err
	}
	loggerConfig, err := env.NewLoggerConfig()
	if err != nil {
		return nil, err
	}
	return &Config{HTTP: httpConfig, Metadata: metadataConfig, Storage: storageConfig, Logger: loggerConfig}, nil
}
