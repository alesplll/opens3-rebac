package env

import (
	"testing"
	"time"
)

func TestTelemetryConfigDefaults(t *testing.T) {
	for _, key := range []string{
		"OTEL_ENABLED", "LOGGER_LEVEL", "LOGGER_AS_JSON", "LOGGER_ENABLE_OLTP",
		"OTEL_SERVICE_NAME", "OTEL_SERVICE_VERSION", "OTEL_ENVIRONMENT",
		"OTEL_EXPORTER_OTLP_ENDPOINT", "OTEL_METRICS_PUSH_TIMEOUT",
	} {
		t.Setenv(key, "")
	}
	t.Setenv("OTEL_ENABLED", "false")
	t.Setenv("LOGGER_LEVEL", "info")
	t.Setenv("LOGGER_AS_JSON", "false")
	t.Setenv("LOGGER_ENABLE_OLTP", "false")
	t.Setenv("OTEL_SERVICE_NAME", "gateway")
	t.Setenv("OTEL_SERVICE_VERSION", "dev")
	t.Setenv("OTEL_ENVIRONMENT", "local")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317")
	t.Setenv("OTEL_METRICS_PUSH_TIMEOUT", "30s")

	cfg, err := NewTelemetryConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Enabled() || cfg.ServiceName() != "gateway" || cfg.ServiceVersion() != "dev" || cfg.PushTimeout() != 30*time.Second {
		t.Fatalf("unexpected telemetry config: %+v", cfg.raw)
	}
}

func TestTelemetryConfigRejectsInvalidPushInterval(t *testing.T) {
	t.Setenv("OTEL_METRICS_PUSH_TIMEOUT", "0s")
	if _, err := NewTelemetryConfig(); err == nil {
		t.Fatal("NewTelemetryConfig() should reject zero push interval")
	}
}
