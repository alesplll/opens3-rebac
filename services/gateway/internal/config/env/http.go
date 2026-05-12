package env

import (
	"errors"
	"net"
	"time"

	"github.com/caarlos0/env/v11"
)

type httpEnvConfig struct {
	Host               string        `env:"HTTP_HOST" envDefault:"0.0.0.0"`
	Port               string        `env:"HTTP_PORT" envDefault:"8080"`
	ReadTimeout        time.Duration `env:"HTTP_READ_TIMEOUT" envDefault:"15s"`
	WriteTimeout       time.Duration `env:"HTTP_WRITE_TIMEOUT" envDefault:"30s"`
	IdleTimeout        time.Duration `env:"HTTP_IDLE_TIMEOUT" envDefault:"60s"`
	ShutdownTimeout    time.Duration `env:"HTTP_SHUTDOWN_TIMEOUT" envDefault:"10s"`
	MaxUploadSizeBytes int64         `env:"MAX_UPLOAD_SIZE_BYTES" envDefault:"5368709120"`
}

type httpConfig struct {
	raw httpEnvConfig
}

func NewHTTPConfig() (*httpConfig, error) {
	var raw httpEnvConfig
	if err := env.Parse(&raw); err != nil {
		return nil, err
	}

	if raw.Port == "" {
		return nil, errors.New("http port is required")
	}

	if raw.MaxUploadSizeBytes <= 0 {
		return nil, errors.New("max upload size must be positive")
	}

	return &httpConfig{raw: raw}, nil
}

func (c *httpConfig) Address() string {
	return net.JoinHostPort(c.raw.Host, c.raw.Port)
}

func (c *httpConfig) Port() string {
	return c.raw.Port
}

func (c *httpConfig) ReadTimeout() time.Duration {
	return c.raw.ReadTimeout
}

func (c *httpConfig) WriteTimeout() time.Duration {
	return c.raw.WriteTimeout
}

func (c *httpConfig) IdleTimeout() time.Duration {
	return c.raw.IdleTimeout
}

func (c *httpConfig) ShutdownTimeout() time.Duration {
	return c.raw.ShutdownTimeout
}

func (c *httpConfig) MaxUploadSizeBytes() int64 {
	return c.raw.MaxUploadSizeBytes
}
