"""
Structured logger — Python analogue of go-kit/logger.

Features:
- JSON or console output to stdout (mirrors zap behaviour)
- Automatic context enrichment: trace_id, span_id from active OTel span
- Optional OTLP log export via gRPC (same collector as traces/metrics)
- Singleton init pattern matching Go version
- Configures root Python logger so stdlib loggers in internal/* produce output

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
# OTel trace context helpers
# ---------------------------------------------------------------------------

def _current_trace_context() -> dict:
    """
    Extract trace_id and span_id from the current active OTel span.
    Returns empty dict if OTel is not installed or no span is active.
    """
    try:
        from opentelemetry import trace
        span = trace.get_current_span()
        ctx = span.get_span_context()
        if ctx is None or not ctx.is_valid:
            return {}
        return {
            "trace_id": format(ctx.trace_id, "032x"),
            "span_id": format(ctx.span_id, "016x"),
        }
    except Exception:
        return {}

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
    """Single-line JSON log records — compatible with Grafana Loki / Elasticsearch."""

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

        # Inject active OTel trace context so every log line carries trace_id/span_id
        payload.update(_current_trace_context())

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

    _LEVEL_COLORS = {
        "DEBUG":    "\033[36m",   # cyan
        "INFO":     "\033[32m",   # green
        "WARNING":  "\033[33m",   # yellow
        "ERROR":    "\033[31m",   # red
        "CRITICAL": "\033[35m",   # magenta
    }
    _RESET = "\033[0m"

    def format(self, record: logging.LogRecord) -> str:
        import datetime

        ts = datetime.datetime.utcfromtimestamp(record.created).strftime("%Y-%m-%dT%H:%M:%S.%f")[:-3] + "Z"
        color = self._LEVEL_COLORS.get(record.levelname, "")
        level_str = f"{color}{record.levelname:<5}{self._RESET}" if color else f"{record.levelname:<5}"
        base = f"{ts}  {level_str}  {record.filename}:{record.lineno}  {record.getMessage()}"

        # Inject trace context inline
        trace_ctx = _current_trace_context()
        if trace_ctx:
            base += f"  trace_id={trace_ctx['trace_id']} span_id={trace_ctx['span_id']}"

        skip = {
            "name", "msg", "args", "levelname", "levelno", "pathname",
            "filename", "module", "exc_info", "exc_text", "stack_info",
            "lineno", "funcName", "created", "msecs", "relativeCreated",
            "thread", "threadName", "processName", "process", "message",
            "taskName",
        }
        extras = {k: v for k, v in record.__dict__.items() if k not in skip}
        if extras:
            kv = "  ".join(f"{k}={v}" for k, v in extras.items())
            base = f"{base}  {kv}"

        if record.exc_info:
            base += "\n" + self.formatException(record.exc_info)

        return base


# ---------------------------------------------------------------------------
# OTLP log handler
# ---------------------------------------------------------------------------

def _build_otlp_handler(cfg: LoggerConfig) -> Optional[logging.Handler]:
    """Create a logging.Handler that ships records to the OTLP collector."""
    try:
        # OTel SDK >= 1.20 uses public opentelemetry.sdk.logs
        # Older versions used private opentelemetry.sdk._logs — try both
        try:
            from opentelemetry.sdk.logs import LoggerProvider, LoggingHandler
            from opentelemetry.sdk.logs.export import BatchLogRecordProcessor
            from opentelemetry._logs import set_logger_provider
        except ImportError:
            from opentelemetry.sdk._logs import LoggerProvider, LoggingHandler  # type: ignore[no-redef]
            from opentelemetry.sdk._logs.export import BatchLogRecordProcessor  # type: ignore[no-redef]
            from opentelemetry._logs import set_logger_provider  # type: ignore[no-redef]

        from opentelemetry.exporter.otlp.proto.grpc._log_exporter import OTLPLogExporter
        from opentelemetry.sdk.resources import Resource, SERVICE_NAME
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

    Configures two loggers:
    - Named logger (cfg.service_name()) — used by py_kit API (logger.info, etc.)
      Gets both stdout + OTLP handlers.
    - Root logger — gets stdout handler only so that stdlib loggers in
      internal/* packages produce visible output in the container console.
    """
    global _logger, _cfg

    if _logger is not None:
        return  # already initialised

    _cfg = cfg
    level = _log_level_map.get(cfg.log_level().lower(), logging.INFO)

    formatter = _JSONFormatter() if cfg.as_json() else _ConsoleFormatter()

    # ── Named logger (py_kit API) ─────────────────────────────────────────
    named = logging.getLogger(cfg.service_name())
    named.setLevel(level)
    named.propagate = False

    stdout_handler = logging.StreamHandler(sys.stdout)
    stdout_handler.setLevel(level)
    stdout_handler.setFormatter(formatter)
    named.addHandler(stdout_handler)

    # OTLP handler (optional) — only on named logger to avoid noise from libs
    if cfg.enable_otlp():
        otlp_handler = _build_otlp_handler(cfg)
        if otlp_handler is not None:
            otlp_handler.setLevel(level)
            named.addHandler(otlp_handler)

    # ── Root logger — makes stdlib loggers in internal/* visible in console ─
    # Named logger has propagate=False so no double-logging for its records.
    root = logging.getLogger()
    root.setLevel(level)
    # Only add handler if root has none yet (avoid duplicates on re-init)
    if not root.handlers:
        root_stdout = logging.StreamHandler(sys.stdout)
        root_stdout.setLevel(level)
        root_stdout.setFormatter(formatter)
        root.addHandler(root_stdout)

    _logger = _Logger(named)


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
