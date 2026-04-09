"""
AuthZ-specific business metrics.

Instruments:
  authz_cache_decisions_total{result="hit|miss"}
      — Redis cache effectiveness.

  authz_decisions_total{action="read|write|delete|...", result="allow|deny"}
      — Authorization decision distribution by action type.

  authz_neo4j_query_duration_seconds{query_type="check|write|delete|read"}
      — Neo4j query latency (separate from gRPC-level histogram).

Usage:
    from internal.metric import authz_metrics

    authz_metrics.init()          # call after metric.init_otel_metrics()

    authz_metrics.record_cache_hit()
    authz_metrics.record_cache_miss()
    authz_metrics.record_decision("read", "allow")
    authz_metrics.record_neo4j_query("check", 0.012)
"""

from __future__ import annotations

import logging

log = logging.getLogger(__name__)

_cache_decisions_counter = None
_decisions_counter = None
_neo4j_query_histogram = None

_NEO4J_LATENCY_BOUNDARIES = [
    0.001, 0.002, 0.005, 0.010, 0.025, 0.050,
    0.100, 0.250, 0.500, 1.000, 2.500, 5.000,
]


def init() -> None:
    """
    Create authz instruments using the global OTel meter from py_kit.
    Must be called after metric.init_otel_metrics().
    """
    global _cache_decisions_counter, _decisions_counter, _neo4j_query_histogram

    try:
        from shared.pkg.py_kit import metric as _metric
        from opentelemetry.sdk.metrics.view import View, ExplicitBucketHistogramAggregation

        meter = _metric.get_meter()
        if meter is None:
            return

        _cache_decisions_counter = meter.create_counter(
            name="authz_cache_decisions_total",
            description="AuthZ Redis cache hit/miss count",
        )

        _decisions_counter = meter.create_counter(
            name="authz_decisions_total",
            description="Authorization decisions by action and result (allow/deny)",
        )

        _neo4j_query_histogram = meter.create_histogram(
            name="authz_neo4j_query_duration_seconds",
            description="Neo4j query latency by query type",
            unit="s",
        )

        log.info("AuthZ metrics initialised")
    except Exception as exc:
        log.warning("Failed to initialise authz metrics: %s", exc)


def record_cache_hit() -> None:
    if _cache_decisions_counter is None:
        return
    _cache_decisions_counter.add(1, attributes={"result": "hit"})


def record_cache_miss() -> None:
    if _cache_decisions_counter is None:
        return
    _cache_decisions_counter.add(1, attributes={"result": "miss"})


def record_decision(action: str, result: str) -> None:
    """
    Record an authorization decision.
    action — e.g. "read", "write", "delete", "admin"
    result — "allow" or "deny"
    """
    if _decisions_counter is None:
        return
    _decisions_counter.add(1, attributes={"action": action, "result": result})


def record_neo4j_query(query_type: str, duration_seconds: float) -> None:
    """
    Record Neo4j query latency.
    query_type — "check", "write", "delete", "read"
    """
    if _neo4j_query_histogram is None:
        return
    _neo4j_query_histogram.record(duration_seconds, attributes={"query_type": query_type})
