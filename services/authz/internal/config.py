"""
Service config — читает env vars один раз при инициализации.
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
    Все значения читаются из env vars один раз при создании объекта.
    """

    def __init__(self):
        # ── Infrastructure ─────────────────────────────────────────────────
        self._neo4j_uri = _get("NEO4J_URI", "bolt://localhost:7687")
        self._neo4j_user = _get("NEO4J_USER", "neo4j")
        self._neo4j_password = _get("NEO4J_PASSWORD", "password123")
        self._redis_host = _get("REDIS_HOST", "localhost")
        self._redis_port = int(_get("REDIS_PORT", "6379"))
        self._kafka_bootstrap = _get("KAFKA_BOOTSTRAP", "localhost:9092")
        self._grpc_port = _get("GRPC_PORT", "50051")
        # ── Observability ──────────────────────────────────────────────────
        self._log_level = _get("LOGGER_LEVEL", "info")
        self._as_json = _get("LOGGER_AS_JSON", "true").lower() in ("true", "1", "yes")
        self._enable_otlp = _get("LOGGER_ENABLE_OTLP", "false").lower() in ("true", "1", "yes")
        self._push_interval_seconds = float(_get("OTEL_METRICS_PUSH_INTERVAL", "60"))
        self._collector_endpoint = _get("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317")
        self._environment = _get("OTEL_ENVIRONMENT", "development")
        self._service_name = _get("OTEL_SERVICE_NAME", "authz")
        self._service_version = _get("OTEL_SERVICE_VERSION", "0.1.0")

    # ── Infrastructure ──────────────────────────────────────────────────────

    def neo4j_uri(self) -> str:
        return self._neo4j_uri

    def neo4j_user(self) -> str:
        return self._neo4j_user

    def neo4j_password(self) -> str:
        return self._neo4j_password

    def redis_host(self) -> str:
        return self._redis_host

    def redis_port(self) -> int:
        return self._redis_port

    def kafka_bootstrap(self) -> str:
        return self._kafka_bootstrap

    def grpc_port(self) -> str:
        return self._grpc_port

    # ── LoggerConfig protocol ────────────────────────────────────────────────

    def log_level(self) -> str:
        return self._log_level

    def as_json(self) -> bool:
        return self._as_json

    def enable_otlp(self) -> bool:
        return self._enable_otlp

    # ── MetricsConfig protocol ───────────────────────────────────────────────

    def push_interval_seconds(self) -> float:
        return self._push_interval_seconds

    # ── TracingConfig protocol ───────────────────────────────────────────────

    def collector_endpoint(self) -> str:
        return self._collector_endpoint

    def environment(self) -> str:
        return self._environment

    # ── Shared (all three protocols) ─────────────────────────────────────────

    def service_name(self) -> str:
        return self._service_name

    def service_version(self) -> str:
        return self._service_version

    def service_environment(self) -> str:
        return self._environment

    def otlp_endpoint(self) -> str:
        return self._collector_endpoint


# Global singleton — создаётся один раз при импорте
cfg = Config()
