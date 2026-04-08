"""Logger config protocol — mirrors go-kit logger.LoggerConfig interface."""
from typing import Protocol


class LoggerConfig(Protocol):
    def log_level(self) -> str:
        """debug | info | warn | error"""
        ...

    def as_json(self) -> bool:
        """Emit JSON lines to stdout."""
        ...

    def enable_otlp(self) -> bool:
        """Send logs to OTLP collector."""
        ...

    def service_name(self) -> str: ...

    def otlp_endpoint(self) -> str:
        """host:port of the OTLP collector gRPC endpoint."""
        ...

    def service_environment(self) -> str:
        """e.g. development | staging | production"""
        ...
