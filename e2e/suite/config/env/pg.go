package env

import (
	"fmt"
	"time"

	cenv "github.com/caarlos0/env/v11"
)

type metadataPGEnvConfig struct {
	DSN      string        `env:"E2E_METADATA_DB_DSN"`
	Host     string        `env:"E2E_METADATA_DB_HOST"`
	Port     string        `env:"E2E_METADATA_DB_PORT"`
	Database string        `env:"E2E_METADATA_DB_NAME"`
	User     string        `env:"E2E_METADATA_DB_USER"`
	Password string        `env:"E2E_METADATA_DB_PASSWORD"`
	SSLMode  string        `env:"E2E_METADATA_DB_SSLMODE" envDefault:"disable"`
	Timeout  time.Duration `env:"E2E_METADATA_DB_TIMEOUT" envDefault:"5s"`
	NeedLog  bool          `env:"E2E_METADATA_DB_LOGGING" envDefault:"false"`
}

type metadataPGConfig struct {
	raw metadataPGEnvConfig
	dsn string
}

func NewMetadataPGConfig() (*metadataPGConfig, error) {
	var raw metadataPGEnvConfig
	if err := cenv.Parse(&raw); err != nil {
		return nil, err
	}

	cfg := &metadataPGConfig{raw: raw}

	if raw.DSN != "" {
		cfg.dsn = raw.DSN
		return cfg, nil
	}

	if raw.Host == "" || raw.Port == "" || raw.Database == "" || raw.User == "" || raw.Password == "" {
		return nil, fmt.Errorf("missing required environment variables for metadata e2e database connection")
	}

	cfg.dsn = fmt.Sprintf(
		"host=%s port=%s dbname=%s user=%s password=%s sslmode=%s",
		raw.Host, raw.Port, raw.Database, raw.User, raw.Password, raw.SSLMode,
	)

	return cfg, nil
}

func (cfg *metadataPGConfig) DSN() string {
	return cfg.dsn
}

func (cfg *metadataPGConfig) Timeout() time.Duration {
	return cfg.raw.Timeout
}

func (cfg *metadataPGConfig) NeedLog() bool {
	return cfg.raw.NeedLog
}
