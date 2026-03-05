"""Kafka producer for audit events."""
import json
import logging
import time
from typing import Dict, Any
from confluent_kafka import Producer
from ..types import Tuple

logger = logging.getLogger(__name__)

class AuditProducer:
    """Produces audit events to Kafka."""

    def __init__(self, bootstrap_servers: str = "localhost:9092", topic: str = "auth-changes"):
        """
        Initialize Kafka producer.

        Args:
            bootstrap_servers: Kafka broker address.
            topic: Audit topic name.
        """
        self.producer = Producer({
            'bootstrap.servers': bootstrap_servers,
        })
        self.topic = topic
        logger.info(f"Audit producer initialized: {bootstrap_servers}/{topic}")

    def send_tuple_event(self, tuple_: Tuple, event_type: str = "tuple_written") -> None:
        """Send tuple change event with correct Redis invalidation hints."""
        event = {
            "event_type": event_type,
            "timestamp": int(1000 * time.time()),  # ms
            "tuple": {
                "subject": tuple_.subject,
                "relation": tuple_.relation,
                "object": tuple_.object,
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

