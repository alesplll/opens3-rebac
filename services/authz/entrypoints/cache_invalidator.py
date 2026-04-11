"""Cache invalidation Kafka consumer."""
import logging
from internal.repositories.cache.redis_cache import RedisDecisionCache
from internal.repositories.cache.invalidation_consumer import CacheInvalidationConsumer

logging.basicConfig(level=logging.INFO)


def main():
    cache = RedisDecisionCache()
    consumer = CacheInvalidationConsumer(redis_cache=cache)
    try:
        consumer.run()
    except KeyboardInterrupt:
        logging.info("Consumer stopped")


if __name__ == "__main__":
    main()
