"""
Graceful shutdown manager — Python analogue of go-kit/closer/closer.go.

Manages a list of cleanup callables that are called in reverse-registration order
when a SIGINT/SIGTERM is received (or when close_all() is called manually).

Usage:
    from shared.pkg.py_kit.closer import configure_default, add_named, close_all
    import signal

    configure_default(signal.SIGINT, signal.SIGTERM)

    add_named("grpc-server",   lambda: server.stop(grace=5).wait())
    add_named("neo4j-driver",  lambda: driver.close())
    add_named("redis-client",  lambda: redis_client.close())
    add_named("otel-metrics",  lambda: metric.shutdown())
    add_named("otel-tracing",  lambda: tracing.shutdown_tracer())

    # Block until signal received:
    closer_instance.wait()

Differences from Go:
  - Cleanup functions are plain callables (no ctx argument) since Python
    does not have the same context cancellation model. If you need timeout
    semantics, wrap the callable with threading.Thread + join(timeout=…).
  - `wait()` blocks the calling thread until shutdown completes.
"""

from __future__ import annotations

import logging
import signal
import sys
import threading
import time
from typing import Callable, Optional, Sequence

log = logging.getLogger(__name__)

_DEFAULT_SHUTDOWN_TIMEOUT = 5.0  # seconds


class Closer:
    """
    Manages graceful shutdown.

    Mirrors the Go Closer struct — register cleanup functions, then either:
      - call `wait()` to block until a signal fires and runs them, or
      - call `close_all()` directly.
    """

    def __init__(self, shutdown_timeout: float = _DEFAULT_SHUTDOWN_TIMEOUT) -> None:
        self._funcs: list[Callable] = []
        self._lock = threading.Lock()
        self._done = threading.Event()
        self._shutdown_timeout = shutdown_timeout
        self._logger = logging.getLogger(__name__)

    def set_logger(self, logger: logging.Logger) -> None:
        self._logger = logger

    def add(self, *funcs: Callable) -> None:
        """Register one or more cleanup callables."""
        with self._lock:
            self._funcs.extend(funcs)

    def add_named(self, name: str, func: Callable) -> None:
        """Register a cleanup callable with a display name for logging."""
        def _named():
            start = time.monotonic()
            self._logger.info("closing %s...", name)
            try:
                func()
                elapsed = time.monotonic() - start
                self._logger.info("%s closed in %.3fs", name, elapsed)
            except Exception as exc:
                elapsed = time.monotonic() - start
                self._logger.error("error closing %s after %.3fs: %s", name, elapsed, exc)

        with self._lock:
            self._funcs.append(_named)

    def close_all(self) -> None:
        """
        Run all registered cleanup functions in reverse order (LIFO).
        Each function runs in its own thread; we wait up to shutdown_timeout.
        Idempotent — subsequent calls are no-ops.
        """
        if self._done.is_set():
            return

        with self._lock:
            funcs = list(reversed(self._funcs))
            self._funcs = []

        if not funcs:
            self._logger.info("no shutdown functions registered")
            self._done.set()
            return

        self._logger.info("starting graceful shutdown (%d functions)...", len(funcs))

        threads = []
        for fn in funcs:
            t = threading.Thread(target=self._safe_call, args=(fn,), daemon=True)
            t.start()
            threads.append(t)

        deadline = time.monotonic() + self._shutdown_timeout
        for t in threads:
            remaining = deadline - time.monotonic()
            if remaining <= 0:
                self._logger.warning("shutdown timeout reached, some resources may not be fully closed")
                break
            t.join(timeout=remaining)

        self._logger.info("graceful shutdown complete")
        self._done.set()

    def wait(self) -> None:
        """Block until close_all() has finished."""
        self._done.wait()

    def _safe_call(self, fn: Callable) -> None:
        try:
            fn()
        except Exception as exc:
            self._logger.error("panic in shutdown function: %s", exc, exc_info=True)

    def _handle_signal(self, signum: int, frame) -> None:
        self._logger.info("signal %d received, starting graceful shutdown...", signum)
        self.close_all()

    def handle_signals(self, *signals: signal.Signals) -> None:
        """Register OS signal handlers that trigger close_all()."""
        for sig in signals:
            signal.signal(sig, self._handle_signal)


# ---------------------------------------------------------------------------
# Global singleton — mirrors Go's globalCloser pattern
# ---------------------------------------------------------------------------

_global_closer = Closer()


def configure(
    logger: Optional[logging.Logger] = None,
    shutdown_timeout: float = _DEFAULT_SHUTDOWN_TIMEOUT,
    *signals: signal.Signals,
) -> Closer:
    """
    Configure the global Closer with a logger, timeout, and OS signals.
    Mirrors Go's closer.Configure().
    """
    global _global_closer
    _global_closer = Closer(shutdown_timeout=shutdown_timeout)
    if logger:
        _global_closer.set_logger(logger)
    if signals:
        _global_closer.handle_signals(*signals)
    return _global_closer


def configure_default(*signals: signal.Signals) -> Closer:
    """
    Configure with defaults (5s timeout, stdlib logger).
    Mirrors Go's closer.ConfigureDefault().
    """
    sigs = signals if signals else (signal.SIGINT, signal.SIGTERM)
    return configure(signals=sigs)


def add(*funcs: Callable) -> None:
    """Register cleanup functions on the global Closer."""
    _global_closer.add(*funcs)


def add_named(name: str, func: Callable) -> None:
    """Register a named cleanup function on the global Closer."""
    _global_closer.add_named(name, func)


def close_all() -> None:
    """Trigger graceful shutdown on the global Closer."""
    _global_closer.close_all()


def wait() -> None:
    """Block until the global Closer finishes shutting down."""
    _global_closer.wait()


def get() -> Closer:
    """Return the global Closer instance."""
    return _global_closer
