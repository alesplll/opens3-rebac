"""Unit tests for the audit event payload, with Kafka replaced by a recorder."""
import json
from pathlib import Path
import sys

REPO_ROOT = Path(__file__).resolve().parents[4]
if str(REPO_ROOT) not in sys.path:
    sys.path.insert(0, str(REPO_ROOT))

from internal.types import Tuple
from internal.repositories.kafka.producer import AuditProducer


class _FakeKafkaProducer:
    """Records payloads instead of sending them to a broker."""

    def __init__(self):
        self.payloads = []

    def produce(self, topic, value, callback=None):
        self.payloads.append(json.loads(value.decode("utf-8")))

    def poll(self, _timeout):
        return 0


def _producer_with(kafka):
    """Wire an AuditProducer to a fake; __init__ would build a real one."""
    producer = AuditProducer.__new__(AuditProducer)
    producer.producer = kafka
    producer.topic = "auth-changes"
    return producer


class TestTupleEventTime:
    def test_uses_the_time_the_database_recorded(self):
        kafka = _FakeKafkaProducer()
        producer = _producer_with(kafka)

        producer.send_tuple_event(
            Tuple("user:alex", "MEMBER_OF", "group:devops"),
            "tuple_written",
            1789317191374,
        )

        assert kafka.payloads[0]["timestamp"] == 1789317191374

    def test_falls_back_to_the_local_clock_when_no_time_is_given(self):
        kafka = _FakeKafkaProducer()
        producer = _producer_with(kafka)

        producer.send_tuple_event(Tuple("user:alex", "MEMBER_OF", "group:devops"))

        assert kafka.payloads[0]["timestamp"] > 0
