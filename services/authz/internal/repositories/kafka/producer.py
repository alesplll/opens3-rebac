"""Kafka producer for audit events."""
import json
import logging
import time
from typing import Optional

from confluent_kafka import Producer
from internal.types import Tuple

logger = logging.getLogger(__name__)


class AuditProducer:
    """Produces audit events to Kafka."""

    def __init__(self, bootstrap_servers: str = "localhost:9092", topic: str = "auth-changes"):
        self.producer = Producer({
            'bootstrap.servers': bootstrap_servers,
        })
        self.topic = topic
        logger.info(f"Audit producer initialized: {bootstrap_servers}/{topic}")

    def send_tuple_event(
        self,
        tuple_: Tuple,
        event_type: str = "tuple_written",
        timestamp_ms: Optional[int] = None,
    ) -> None:
        """Send tuple change event with correct Redis invalidation hints.

        `timestamp_ms` is the time the graph recorded the change, so the event
        and the edge it describes carry one time from one clock. It falls back
        to the local clock only for a caller that has no such time.
        """
        event = {
            "event_type": event_type,
            "timestamp": int(1000 * time.time()) if timestamp_ms is None else timestamp_ms,  # ms
            "tuple": {
                "subject": tuple_.subject,
                "relation": tuple_.relation,
                "object": tuple_.object,
                # None when the caller did not identify the initiator.
                "actor": tuple_.actor,
            },
            # Patterns match auth_decision:{subject}:{action}:{object}
            "invalidation_hints": [
                f"auth_decision:{tuple_.subject}:*:{tuple_.object}",
                f"auth_decision:*:*:{tuple_.object}",
            ],
        }
        self.producer.produce(
            topic=self.topic,
            value=json.dumps(event).encode('utf-8'),
            callback=self._delivery_report,
        )
        self.producer.poll(0)

    def send_decision_event(self, subject: str, action: str, object_: str, allowed: bool) -> None:
        """Send access decision audit event (ACCESS_GRANTED / ACCESS_DENIED)."""
        event = {
            "event_type": "ACCESS_GRANTED" if allowed else "ACCESS_DENIED",
            "timestamp": int(1000 * time.time()),  # ms
            "subject": subject,
            "action": action,
            "object": object_,
        }
        self.producer.produce(
            topic=self.topic,
            value=json.dumps(event).encode('utf-8'),
            callback=self._delivery_report,
        )
        self.producer.poll(0)

    def _delivery_report(self, err, msg):
        """Delivery callback."""
        if err is not None:
            logger.error(f"Message delivery failed: {err}")
        else:
            logger.debug(f"Message delivered to {msg.topic()} [{msg.partition()}]")
