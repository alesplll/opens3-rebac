"""
Service config — читает env vars, реализует протоколы py_kit.
"""
import os


def _require(name: str) -> str:
    val = os.environ.get(name)
    if not val:
        raise RuntimeError(f"Required env var {name!r} is not set")
    return val


def _get(name: str, default: str) -> str:
    return os.environ.get(name, default)


class Config:
    """
    Единый конфиг сервиса: инфраструктура + observability.
    Все значения читаются из env vars при каждом обращении.
    """

    # ── Infrastructure ─────────────────────────────────────────────────────

    def neo4j_uri(self) -> str:
        return _get("NEO4J_URI", "bolt://localhost:7687")

    def neo4j_user(self) -> str:
        return _get("NEO4J_USER", "neo4j")

    def neo4j_password(self) -> str:
        return _get("NEO4J_PASSWORD", "password123")

    def redis_host(self) -> str:
        return _get("REDIS_HOST", "localhost")

    def redis_port(self) -> int:
        return int(_get("REDIS_PORT", "6379"))

    def kafka_bootstrap(self) -> str:
        return _get("KAFKA_BOOTSTRAP", "localhost:9092")

    def grpc_port(self) -> str:
        return _get("GRPC_PORT", "50051")

    # ── LoggerConfig protocol ──────────────────────────────────────────────

    def log_level(self) -> str:
        return _get("LOGGER_LEVEL", "info")

    def as_json(self) -> bool:
        return _get("LOGGER_AS_JSON", "true").lower() in ("true", "1", "yes")

    def enable_otlp(self) -> bool:
        return _get("LOGGER_ENABLE_OTLP", "false").lower() in ("true", "1", "yes")

    # ── MetricsConfig protocol ─────────────────────────────────────────────

    def push_interval_seconds(self) -> float:
        return float(_get("OTEL_METRICS_PUSH_INTERVAL", "60"))

    # ── TracingConfig protocol ─────────────────────────────────────────────

    def collector_endpoint(self) -> str:
        return _get("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317")

    def environment(self) -> str:
        return _get("OTEL_ENVIRONMENT", "development")

    # ── Shared (all three protocols) ───────────────────────────────────────

    def service_name(self) -> str:
        return _get("OTEL_SERVICE_NAME", "authz")

    def service_version(self) -> str:
        return _get("OTEL_SERVICE_VERSION", "0.1.0")

    def service_environment(self) -> str:
        return self.environment()

    def otlp_endpoint(self) -> str:
        return self.collector_endpoint()


# Global singleton — создаётся один раз при импорте
cfg = Config()
