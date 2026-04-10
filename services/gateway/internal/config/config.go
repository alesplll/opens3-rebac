package config

import (
	"os"

	"github.com/alesplll/opens3-rebac/services/gateway/internal/config/env"
	"github.com/joho/godotenv"
)

var appConfig *config

type config struct {
	HTTP        HTTPConfig
	Logger      LoggerConfig
	Tracing     TracingConfig
	Metrics     MetricsConfig
	RateLimiter RateLimiterConfig
	AuthZClient GRPCClientConfig
	Metadata    GRPCClientConfig
	Storage     GRPCClientConfig
	JWT         JWTConfig
}

func Load(path ...string) error {
	err := godotenv.Load(path...)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	httpCfg, err := env.NewHTTPConfig()
	if err != nil {
		return err
	}

	loggerCfg, err := env.NewLoggerConfig()
	if err != nil {
		return err
	}

	tracingCfg, err := env.NewTracingConfig()
	if err != nil {
		return err
	}

	metricsCfg, err := env.NewMetricsConfig()
	if err != nil {
		return err
	}

	rateLimiterCfg, err := env.NewRateLimiterConfig()
	if err != nil {
		return err
	}

	authzCfg, err := env.NewAuthZClientConfig()
	if err != nil {
		return err
	}

	metadataCfg, err := env.NewMetadataClientConfig()
	if err != nil {
		return err
	}

	storageCfg, err := env.NewStorageClientConfig()
	if err != nil {
		return err
	}

	jwtCfg, err := env.NewJWTConfig()
	if err != nil {
		return err
	}

	appConfig = &config{
		HTTP:        httpCfg,
		Logger:      loggerCfg,
		Tracing:     tracingCfg,
		Metrics:     metricsCfg,
		RateLimiter: rateLimiterCfg,
		AuthZClient: authzCfg,
		Metadata:    metadataCfg,
		Storage:     storageCfg,
		JWT:         jwtCfg,
	}

	return nil
}

func AppConfig() *config {
	return appConfig
}
