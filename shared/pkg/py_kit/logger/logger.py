"""
Structured logger — Python analogue of go-kit/logger.

Features:
- JSON or console output to stdout (mirrors zap behaviour)
- Automatic context enrichment: trace_id, user_id from context dict
- Optional OTLP log export via gRPC (same collector as traces/metrics)
- Singleton init pattern matching Go version

Usage:
    from shared.pkg.py_kit.logger import init, info, with_context
    from shared.pkg.py_kit.logger.config import LoggerConfig

    class MyCfg:
        def log_level(self): return "info"
        def as_json(self): return True
        def enable_otlp(self): return True
        def service_name(self): return "authz"
        def otlp_endpoint(self): return "otel-collector:4317"
        def service_environment(self): return "development"

    init(MyCfg())
    info({"trace_id": "abc"}, "Permission check", action="read")
"""

from __future__ import annotations

import logging
import sys
from typing import Any, Optional

from .config import LoggerConfig

# ---------------------------------------------------------------------------
# Internal state (mirrors Go's globalLogger + initOnce)
# ---------------------------------------------------------------------------

_logger: Optional["_Logger"] = None
_cfg: Optional[LoggerConfig] = None
_log_level_map = {
    "debug": logging.DEBUG,
    "info": logging.INFO,
    "warn": logging.WARNING,
    "warning": logging.WARNING,
    "error": logging.ERROR,
    "fatal": logging.CRITICAL,
}


# ---------------------------------------------------------------------------
# JSON formatter
# ---------------------------------------------------------------------------

class _JSONFormatter(logging.Formatter):
    """Single-line JSON log records — compatible with Grafana Loki."""

    def format(self, record: logging.LogRecord) -> str:
        import json
        import datetime

        payload: dict[str, Any] = {
            "timestamp": datetime.datetime.utcfromtimestamp(record.created).isoformat() + "Z",
            "level": record.levelname,
            "logger": record.name,
            "caller": f"{record.filename}:{record.lineno}",
            "message": record.getMessage(),
        }

        # Extra fields attached via LoggerAdapter or record.__dict__
        skip = {
            "name", "msg", "args", "levelname", "levelno", "pathname",
            "filename", "module", "exc_info", "exc_text", "stack_info",
            "lineno", "funcName", "created", "msecs", "relativeCreated",
            "thread", "threadName", "processName", "process", "message",
            "taskName",
        }
        for key, val in record.__dict__.items():
            if key not in skip:
                payload[key] = val

        if record.exc_info:
            payload["stacktrace"] = self.formatException(record.exc_info)

        return json.dumps(payload, ensure_ascii=False, default=str)


class _ConsoleFormatter(logging.Formatter):
    """Human-readable format: timestamp level caller message [key=val ...]"""

    def format(self, record: logging.LogRecord) -> str:
        import datetime

        ts = datetime.datetime.utcfromtimestamp(record.created).strftime("%Y-%m-%dT%H:%M:%S.%f")[:-3] + "Z"
        base = f"{ts}\t{record.levelname:<5}\t{record.filename}:{record.lineno}\t{record.getMessage()}"

        skip = {
            "name", "msg", "args", "levelname", "levelno", "pathname",
            "filename", "module", "exc_info", "exc_text", "stack_info",
            "lineno", "funcName", "created", "msecs", "relativeCreated",
            "thread", "threadName", "processName", "process", "message",
            "taskName",
        }
        extras = {k: v for k, v in record.__dict__.items() if k not in skip}
        if extras:
            kv = "\t".join(f"{k}={v}" for k, v in extras.items())
            base = f"{base}\t{kv}"

        if record.exc_info:
            base += "\n" + self.formatException(record.exc_info)

        return base


# ---------------------------------------------------------------------------
# OTLP log handler
# ---------------------------------------------------------------------------

def _build_otlp_handler(cfg: LoggerConfig) -> Optional[logging.Handler]:
    """Create a logging.Handler that ships records to the OTLP collector."""
    try:
        from opentelemetry.sdk._logs import LoggerProvider
        from opentelemetry.sdk._logs.export import BatchLogRecordProcessor
        from opentelemetry.exporter.otlp.proto.grpc._log_exporter import OTLPLogExporter
        from opentelemetry.sdk.resources import Resource, SERVICE_NAME
        from opentelemetry._logs import set_logger_provider
        from opentelemetry.sdk._logs import LoggingHandler  # type: ignore[attr-defined]
    except ImportError:
        logging.getLogger(__name__).warning(
            "opentelemetry packages not installed — OTLP log export disabled. "
            "Install: opentelemetry-sdk opentelemetry-exporter-otlp-proto-grpc"
        )
        return None

    resource = Resource.create({
        SERVICE_NAME: cfg.service_name(),
        "deployment.environment": cfg.service_environment(),
    })

    exporter = OTLPLogExporter(
        endpoint=cfg.otlp_endpoint(),
        insecure=True,
    )

    provider = LoggerProvider(resource=resource)
    provider.add_log_record_processor(BatchLogRecordProcessor(exporter))
    set_logger_provider(provider)

    handler = LoggingHandler(logger_provider=provider)
    return handler


# ---------------------------------------------------------------------------
# Logger wrapper
# ---------------------------------------------------------------------------

class _Logger:
    """
    Thin wrapper around a stdlib logger that adds context enrichment.
    Context is a plain dict: {"trace_id": "...", "user_id": "..."}
    """

    def __init__(self, underlying: logging.Logger) -> None:
        self._log = underlying

    def _emit(self, level: int, ctx: dict, msg: str, **fields: Any) -> None:
        extra = {**ctx, **fields}
        self._log.log(level, msg, extra=extra, stacklevel=3)

    def debug(self, ctx: dict, msg: str, **fields: Any) -> None:
        self._emit(logging.DEBUG, ctx, msg, **fields)

    def info(self, ctx: dict, msg: str, **fields: Any) -> None:
        self._emit(logging.INFO, ctx, msg, **fields)

    def warn(self, ctx: dict, msg: str, **fields: Any) -> None:
        self._emit(logging.WARNING, ctx, msg, **fields)

    def error(self, ctx: dict, msg: str, **fields: Any) -> None:
        self._emit(logging.ERROR, ctx, msg, **fields)

    def fatal(self, ctx: dict, msg: str, **fields: Any) -> None:
        self._emit(logging.CRITICAL, ctx, msg, **fields)
        sys.exit(1)

    def with_fields(self, **fields: Any) -> "_BoundLogger":
        return _BoundLogger(self, {}, fields)

    def with_context(self, ctx: dict) -> "_BoundLogger":
        return _BoundLogger(self, ctx, {})


class _BoundLogger:
    """Logger pre-loaded with context + static fields — mirrors zap.With / logger.WithContext."""

    def __init__(self, base: _Logger, ctx: dict, static: dict) -> None:
        self._base = base
        self._ctx = ctx
        self._static = static

    def debug(self, msg: str, **fields: Any) -> None:
        self._base.debug({**self._ctx, **self._static, **fields}, msg)

    def info(self, msg: str, **fields: Any) -> None:
        self._base.info({**self._ctx, **self._static, **fields}, msg)

    def warn(self, msg: str, **fields: Any) -> None:
        self._base.warn({**self._ctx, **self._static, **fields}, msg)

    def error(self, msg: str, **fields: Any) -> None:
        self._base.error({**self._ctx, **self._static, **fields}, msg)

    def fatal(self, msg: str, **fields: Any) -> None:
        self._base.fatal({**self._ctx, **self._static, **fields}, msg)


# ---------------------------------------------------------------------------
# Public API — mirrors go-kit logger package-level functions
# ---------------------------------------------------------------------------

def init(cfg: LoggerConfig) -> None:
    """
    Initialise the global logger. Call once at startup.
    Idempotent — subsequent calls are no-ops (mirrors Go's sync.Once).
    """
    global _logger, _cfg

    if _logger is not None:
        return  # already initialised

    _cfg = cfg
    level = _log_level_map.get(cfg.log_level().lower(), logging.INFO)

    root = logging.getLogger(cfg.service_name())
    root.setLevel(level)
    root.propagate = False

    # Stdout handler
    stdout_handler = logging.StreamHandler(sys.stdout)
    stdout_handler.setLevel(level)
    if cfg.as_json():
        stdout_handler.setFormatter(_JSONFormatter())
    else:
        stdout_handler.setFormatter(_ConsoleFormatter())
    root.addHandler(stdout_handler)

    # OTLP handler (optional)
    if cfg.enable_otlp():
        otlp_handler = _build_otlp_handler(cfg)
        if otlp_handler is not None:
            otlp_handler.setLevel(level)
            root.addHandler(otlp_handler)

    _logger = _Logger(root)


def set_nop_logger() -> None:
    """Replace global logger with a no-op (useful in tests)."""
    global _logger

    nop = logging.getLogger("nop")
    nop.addHandler(logging.NullHandler())
    nop.propagate = False
    _logger = _Logger(nop)


def get_logger() -> _Logger:
    """Return the global logger instance. Raises if init() was not called."""
    if _logger is None:
        raise RuntimeError("py-kit logger not initialised — call logger.init(cfg) first")
    return _logger


def with_context(ctx: dict) -> _BoundLogger:
    return get_logger().with_context(ctx)


# Package-level convenience functions — mirror Go's logger.Info(ctx, msg, ...) etc.

def debug(ctx: dict, msg: str, **fields: Any) -> None:
    get_logger().debug(ctx, msg, **fields)


def info(ctx: dict, msg: str, **fields: Any) -> None:
    get_logger().info(ctx, msg, **fields)


def warn(ctx: dict, msg: str, **fields: Any) -> None:
    get_logger().warn(ctx, msg, **fields)


def error(ctx: dict, msg: str, **fields: Any) -> None:
    get_logger().error(ctx, msg, **fields)


def fatal(ctx: dict, msg: str, **fields: Any) -> None:
    get_logger().fatal(ctx, msg, **fields)


def sync() -> None:
    """Flush all log handlers — call at shutdown (mirrors zap Sync)."""
    if _logger is not None:
        for handler in _logger._log.handlers:
            try:
                handler.flush()
            except Exception:
                pass
