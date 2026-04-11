"""Dependency container — wires all service objects from config."""
from internal.config import Config
from internal.repositories.neo4j.store import Neo4jStore
from internal.repositories.cache.redis_cache import RedisDecisionCache
from internal.repositories.kafka.producer import AuditProducer
from internal.permission.service import PermissionService


class Container:
    def __init__(self, cfg: Config):
        self.neo4j_store = Neo4jStore(
            uri=cfg.neo4j_uri(),
            user=cfg.neo4j_user(),
            password=cfg.neo4j_password(),
        )
        self.cache = RedisDecisionCache(
            host=cfg.redis_host(),
            port=cfg.redis_port(),
        )
        self.audit_producer = AuditProducer(cfg.kafka_bootstrap())
        self.rebac = PermissionService(
            store=self.neo4j_store,
            cache=self.cache,
            audit_producer=self.audit_producer,
        )
