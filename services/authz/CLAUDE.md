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
- текущий producer отправляет audit и tuple events в один topic `auth-changes`, заданный default конструктора;
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

## Навигация и границы слоёв

- `entrypoints/server/main.py`: bootstrap telemetry/container, health и reflection.
- `entrypoints/server/servicer.py`: enum mapping и protobuf transport.
- `internal/config.py`: singleton `cfg`, переменные читаются при импорте.
- `internal/container.py`: Neo4jStore, RedisDecisionCache, AuditProducer, PermissionService.
- `internal/types.py`: доменный Tuple.
- `internal/permission/interfaces.py`: GraphStore Protocol.
- `internal/permission/service.py`: cache → graph → cache write → audit.
- `internal/repositories/neo4j/{schema,store}.py`: hierarchy и Cypher.
- `internal/repositories/cache/`: DecisionCache, Redis и invalidation consumer.
- `internal/repositories/kafka/producer.py`: формирование events и async delivery.
- `internal/metric/`: service metrics.

`internal` — соглашение организации Python, не запрет импорта. Не переносите
transport enum mapping в Neo4jStore; правила обхода должны тестироваться отдельно.

## Локальная разработка

Полный самостоятельный setup с `.venv`, editable install и зависимостями описан в
[README](README.md#запуск). Запускайте из каталога сервиса:

```bash
python -m entrypoints.server.main
python -m entrypoints.cache_invalidator
```

Это два **отдельных терминала/процесса**, а не последовательный запуск в одном
терминале. Consumer entrypoint пока использует localhost constructor defaults и
не читает адреса из Config. Простой перенос этого процесса в контейнер не работает
как полноценная конфигурация invalidation.

После изменения proto используйте корневые targets `make generate-authz-go` и
`make generate-authz-py` после `make install-deps`. Generated Python лежит в
`shared/pkg/py/authz/v1`. Локальные изменения пользователя в `proto/generate.sh`
не следует перезаписывать при редактировании документации.

## Сценарии и события

Cache hit включает сохранённый DENY: проверяйте `cached is not None`, а не
truthiness. При miss вызывается graph check; TTL 30 секунд отсчитывается от set.
Audit вызывается и при cache hit. В текущем wiring и tuple, и decision events идут
в `auth-changes`. Decision event не содержит отдельного `allowed`/`from_cache`;
решение выражено в event_type. `produce()` может синхронно выбросить исключение,
а delivery callback только логирует ошибку — это разные failure paths.

## Тестовые соглашения

В unit tests мокайте GraphStore, DecisionCache и AuditProducer через
`unittest.mock.MagicMock`. Health tests проверяют `store.health()` и
`cache.health()`, а не приватные `driver`/`_client`. Разделяйте allow/deny, cache
hit/miss, неверные enums, ошибки зависимостей и отсутствие лишних вызовов.

```bash
python -m pytest tests/unit -v --cov=internal --cov=entrypoints --cov-report=term-missing
python -m pytest tests/integration -v -m integration
```

Graph integration требует Neo4j; skip из-за недоступной БД не считать успешной
проверкой query. Для нового revoke протокола нужны group-membership и concurrent
check/invalidation cases, а не только удаление одного прямого разрешения.
