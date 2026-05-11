package env

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
)

type pgEnvConfig struct {
	DSN      string        `env:"PG_DSN"`
	Host     string        `env:"PG_HOST"`
	Port     string        `env:"PG_PORT_INNER"`
	Database string        `env:"PG_DATABASE_NAME"`
	User     string        `env:"PG_USER"`
	Password string        `env:"PG_PASSWORD"`
	SSLMode  string        `env:"PG_SSL_MODE" envDefault:"disable"`
	Timeout  time.Duration `env:"PG_TIMEOUT" envDefault:"5s"`
	NeedLog  bool          `env:"PG_LOGGING" envDefault:"false"`
}

type pgConfig struct {
	raw pgEnvConfig
	dsn string
}

func NewPGConfig() (*pgConfig, error) {
	var raw pgEnvConfig
	if err := env.Parse(&raw); err != nil {
		return nil, err
	}

	cfg := &pgConfig{raw: raw}

	if raw.DSN != "" {
		cfg.dsn = raw.DSN
		return cfg, nil
	}

	if raw.Host == "" || raw.Port == "" || raw.Database == "" ||
		raw.User == "" || raw.Password == "" {
		return nil, fmt.Errorf("missing required environment variables for database connection (Tip: you can just define PG_DSN in environment)")
	}

	cfg.dsn = fmt.Sprintf(
		"host=%s port=%s dbname=%s user=%s password=%s sslmode=%s",
		raw.Host, raw.Port, raw.Database, raw.User, raw.Password, raw.SSLMode,
	)

	return cfg, nil
}

func (cfg *pgConfig) DSN() string {
	return cfg.dsn
}

func (cfg *pgConfig) Timeout() time.Duration {
	return cfg.raw.Timeout
}

func (cfg *pgConfig) NeedLog() bool {
	return cfg.raw.NeedLog
}
