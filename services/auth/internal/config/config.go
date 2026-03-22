package config

import (
	"os"

	"github.com/alesplll/opens3-rebac/services/auth/internal/config/env"
	"github.com/joho/godotenv"
)

// appConfig holds the global application configuration instance.
var appConfig *config

// config represents the complete application configuration.
type config struct {
	Logger      LoggerConfig
	GRPC        GRPCConfig
	UserClient  GRPCConfig
	Redis       RedisConfig
	JWT         JWTConfig
	Security    SecurityConfig
	Tracing     TracingConfig
	Metrics     MetricsConfig
	RateLimiter RateLimiterConfig
}

// Load reads environment variables from .env file(s) and initializes the application configuration.
// The function ignores file-not-found errors, allowing configuration to be loaded
// from system environment variables when .env file is absent.
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

	userClientCfg, err := env.NewUserGRPCConfig()
	if err != nil {
		return err
	}

	redisCfg, err := env.NewRedisConfig()
	if err != nil {
		return err
	}

	jwtCfg, err := env.NewJWTConfig()
	if err != nil {
		return err
	}

	securityCfg, err := env.NewSecurityConfig()
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
		UserClient:  userClientCfg,
		Redis:       redisCfg,
		JWT:         jwtCfg,
		Security:    securityCfg,
		Metrics:     metricsCfg,
		Tracing:     tracingCfg,
		RateLimiter: rateLimiterCfg,
	}

	return nil
}

// AppConfig returns the global application configuration instance.
// Panics if called before Load().
func AppConfig() *config {
	return appConfig
}
