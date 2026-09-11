# AuthZ context for assistants

Перед изменением сервиса прочитайте [README.md](README.md) и корневой
[AGENTS.md](../../AGENTS.md).

Ключевые ограничения текущей реализации:

- protobuf package — `opens3.authz.v1`; enums передаются числами/именами protobuf,
  а не произвольными строками;
- публичные relations — MEMBER_OF, HAS_PERMISSION, PARENT_OF; OWNER_OF legacy и
  не принимается public API;
- graph-check поддерживает group membership и direct permission на целевом
  resource, но не PARENT_OF inheritance;
- permission hierarchy: admin > delete > create > write > read;
- decision TTL задан в `internal/permission/service.py` как 30 секунд;
- текущий producer отправляет audit и tuple events в один configured topic;
- основной server process не запускает invalidation consumer;
- invalidation не даёт строгого immediate revoke для group-derived decisions.

Не описывайте cache invalidation как транзакционную: Neo4j mutation, Kafka publish
и Redis cleanup выполняются раздельно. Для усиления гарантии нужны epochs или
dependency-aware invalidation, durable publication и race tests.

Команды:

```bash
cd services/authz
python3 -m pytest tests/unit -v
python3 -m pytest tests/integration -v -m integration
```

Docker build выполняется из корня:

```bash
docker build -f services/authz/Dockerfile .
```
