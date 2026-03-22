package config

import (
	"os"

	"github.com/alesplll/opens3-rebac/services/storage/internal/config/env"
	"github.com/joho/godotenv"
)

var appConfig *config

type config struct {
	Logger      LoggerConfig
	GRPC        GRPCConfig
	Storage     StorageConfig
	Tracing     TracingConfig
	Metrics     MetricsConfig
	RateLimiter RateLimiterConfig
}

func Load(path ...string) error {
	err := godotenv.Load(path...)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	loggerCfg, err := env.NewLoggerConfig()
	if err != nil {
		return err
	}

	grpcCfg, err := env.NewGRPCConfig()
	if err != nil {
		return err
	}

	storageCfg, err := env.NewStorageConfig()
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

	appConfig = &config{
		Logger:      loggerCfg,
		GRPC:        grpcCfg,
		Storage:     storageCfg,
		Tracing:     tracingCfg,
		Metrics:     metricsCfg,
		RateLimiter: rateLimiterCfg,
	}

	return nil
}

func AppConfig() *config {
	return appConfig
}
