# Quota Service

Rust 1.95 gRPC-сервис учёта лимитов пользователей и buckets. Горячее состояние
находится в памяти (`DashMap`), Redis используется для периодического сохранения.

## Назначение и возможности

Учёт bytes/objects/buckets, резервирование квоты, компенсация расхода, настройка лимитов и удаление subject. Сервис не хранит байты и не проверяет права AuthZ.

## Архитектура и зависимости

```text
gRPC handler → QuotaService → MemoryCache (DashMap)
                    │               │
                    └──── flush ────┴──> RedisRepository
```

Проверка и резервирование выполняются в памяти без repository I/O на hot path.
Фоновая задача сохраняет dirty entries в Redis. Limits записываются через
repository реже, usage меняется на каждом reserve/compensation.

## Структура проекта

```text
src/main.rs                 process entrypoint
src/lib.rs                  public modules
src/app.rs                  wiring, background flush, lifecycle
src/config.rs               environment parsing
src/domain/                 quota types и errors
src/cache/memory.rs         in-memory usage/limits
src/repository/             trait и Redis implementation
src/service/quota.rs        reserve/update business logic
src/transport/grpc.rs       protobuf mapping и gRPC handler
src/metrics.rs              OpenTelemetry metrics
tests/                      gRPC и Redis integration tests
```

## API

Source of truth: `shared/api/quota/v1/quota.proto`.

| RPC | Назначение |
|---|---|
| `CheckQuota` | проверить и сразу зарезервировать положительную дельту |
| `UpdateUsage` | применить самостоятельную дельту/компенсацию |
| `GetUsage` | получить текущий usage |
| `SetQuota` / `GetQuota` | управлять limits |
| `DeleteSubject` | удалить usage и limits subject |
| `HealthCheck` | проверить доступность repository |

### Правильный write flow

```text
CheckQuota(+delta)
  denied → операцию не выполнять
  allowed → резерв уже учтён
      операция успешна → ничего не добавлять повторно
      операция неуспешна → UpdateUsage(-delta) как компенсация
```

Вызов `UpdateUsage(+delta)` после allowed `CheckQuota(+delta)` начислит расход
дважды. Текущий контракт не содержит operation/reservation ID, поэтому timeout и
retry мутации не являются exactly-once. Для надёжного production flow нужны
стабильный operation ID, сохранённый terminal result и reconciliation.

### Пример жизненного цикла

```text
CreateBucket: CheckQuota(buckets=+1) → выполнить create
PutObject:    CheckQuota(bytes=+size, objects=+1) → выполнить write
ошибка write: UpdateUsage(bytes=-size, objects=-1)
DeleteObject: UpdateUsage(bytes=-size, objects=-1)
```

Если после allowed reserve операция завершилась успешно, положительный
`UpdateUsage` для той же дельты не вызывается.

## Данные и основные сценарии

### Границы консистентности

Операция над одной записью DashMap синхронизирована внутри одного процесса.
Резерв пользователя и bucket — два шага с компенсацией user при отказе bucket.
Это не распределённая транзакция. Несколько активных Quota replicas держат
независимое состояние и не обеспечивают единый строгий лимит, поэтому текущий
deployment должен иметь одного active writer.

`DashMap::new()` сам выбирает число shards; это не фиксированные 256 shards и не
lock-free структура. Оценки nanoseconds для map operation нельзя переносить на
полный gRPC request без benchmark.

### Persistence

При старте service загружает usage и limits из Redis в память; ошибка загрузки
останавливает startup. SetQuota меняет cache и синхронно сохраняет limits в Redis;
если сохранение не удалось, cache уже изменён. GetUsage неизвестного subject
возвращает нули. GetQuota возвращает сохранённый limit или отсутствие записи;
defaults при CheckQuota и отсутствие настроенного limit — разные ситуации.

Фоновый task вызывает flush с интервалом `REDIS_FLUSH_INTERVAL_MS`. В текущем
development `.env` это 500 ms, default кода — 1000 ms. Интервал не является
гарантированной верхней границей потери: между изменением памяти, flush и
persistence Redis остаются окна сбоя.

Development Redis в Compose запущен без AOF. После перезапуска Redis persisted
quota data может исчезнуть. Включение AOF уменьшает одно из окон, но не делает
весь протокол синхронно durable.

При неуспешном `flush_usage` snapshot keys снова помечаются dirty и повторяются в
следующем цикле. Concurrent delete остаётся отдельным lifecycle race: ранее
взятый snapshot может быть записан после `DeleteSubject`, поэтому для строгой
семантики нужны generation/tombstone либо сериализация операций.

## Конфигурация

```dotenv
GRPC_PORT=50055
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_DB=1
REDIS_FLUSH_INTERVAL_MS=500
DEFAULT_USER_BYTES_LIMIT=10737418240
DEFAULT_USER_OBJECTS_LIMIT=-1
DEFAULT_USER_BUCKETS_LIMIT=100
DEFAULT_BUCKET_BYTES_LIMIT=-1
DEFAULT_BUCKET_OBJECTS_LIMIT=-1
```

Значение 100 buckets — собственный default проекта, не текущий AWS quota. Размеры
и HTTP mapping внешнего S3 API определяет будущий Gateway; 507 — расширение
проекта, а не универсальный S3 error contract.

Для локальной Rust-сборки нужен `protoc`, потому что `build.rs` компилирует proto.
Dockerfile устанавливает его через `protobuf-dev`.

Полный набор параметров определён в `src/config.rs`. При запуске внутри Compose
используется container address Redis; при запуске с хоста — опубликованный host
address.

### Дополнительные переменные

| Переменная | Default кода / назначение |
|---|---|
| `REDIS_PASSWORD` | не задан; используется при формировании URL |
| `SERVICE_NAME`, `ENVIRONMENT` | `quota`, `development` |
| `LOG_LEVEL`, `LOG_JSON` | `info`, `true` |
| `ENABLE_OTLP`, `OTLP_ENDPOINT` | `false`, `http://otel-collector:4317` |

Все лимиты перечислены выше; `-1` означает unlimited. Redis DB 1 отделяет Quota
от AuthZ DB 0. Параметры загружаются один раз в `OnceLock`.

## Запуск

### Локально

Из корня репозитория, с установленным Rust toolchain и `protoc`:

```bash
docker compose up -d redis
docker compose --profile services stop quota
cargo run -p quota-service
```

Запуск из корня использует defaults и окружение корня: Redis localhost, DB 1,
flush 1000 ms. Чтобы читать сервисный `.env`, запускайте Cargo из `services/quota`;
в нём flush 500 ms. Не запускайте контейнер и local process одновременно на 50055.

### Docker

Из корня:

```bash
docker compose --profile services up --build -d quota
docker compose logs -f quota
```

Compose переопределяет Redis address на `redis`, включает OTLP и задаёт
`http://otel-collector:4317`. Collector запускается через `make up-observability`.

## Примеры использования

Пример резервирует 3 байта и один объект, затем освобождает резерв без записи файла.
Выполняйте компенсацию только после `allowed=true`:

```bash
grpcurl -plaintext -d '{"subject_id":"user:docs-demo","bucket_id":"bucket:docs-demo","delta":{"bytes":"3","objects":"1"}}' \
  localhost:50055 opens3.quota.v1.QuotaService/CheckQuota
grpcurl -plaintext -d '{"subject_id":"user:docs-demo"}' \
  localhost:50055 opens3.quota.v1.QuotaService/GetUsage
grpcurl -plaintext -d '{"subject_id":"user:docs-demo","bucket_id":"bucket:docs-demo","delta":{"bytes":"-3","objects":"-1"}}' \
  localhost:50055 opens3.quota.v1.QuotaService/UpdateUsage
```

`allowed=false` и `code` описывают отказ квоты в обычном ответе. При timeout нельзя
вслепую повторить reserve или compensation: результат первой мутации неизвестен.
В SetQuota указывайте все три лимита; пропущенный числовой protobuf field равен
нулю, для unlimited нужен `-1`.

## Наблюдаемость и диагностика

```bash
grpcurl -plaintext localhost:50055 opens3.quota.v1.QuotaService/HealthCheck
```

RPC проверяет Redis. При сбое flush смотрите `quota_redis_flush_errors_total`,
`quota_redis_flush_total`, `quota_redis_flush_entries` и `quota_redis_flush_duration_seconds`
из [src/metrics.rs](src/metrics.rs). Dashboard: [quota.json](../../infra/otel/grafana/dashboards/quota.json).
После неуспешного flush dirty keys сохраняются для повтора; это не защищает от падения процесса до сохранения памяти.

## Разработка и тесты

Отдельные suites (из корня; Redis DB 15 должен быть выделен для тестов):

```bash
cargo test -p quota-service --test grpc
TEST_REDIS_URL=redis://localhost:6379/15 cargo test -p quota-service --test redis -- --include-ignored
```

Redis tests помечены `#[ignore]`; обычный `cargo test` их не запускает. Даже с
`--include-ignored` они возвращаются без проверки, если подключение не удалось:
проверяйте доступность Redis, а не только итоговый exit code.
Rust stubs создаёт `build.rs` при Cargo build; shared Go/Python clients обновляются
через `make generate-quota-go` / `make generate-quota-py` после `make install-deps`.

```bash
cargo fmt -p quota-service -- --check
cargo clippy -p quota-service -- -D warnings
cargo test -p quota-service
```

Redis integration tests запускаются согласно `.github/workflows/quota-ci.yml` и
требуют доступный Redis. gRPC tests поднимают in-process Tonic server.

## Ограничения и дальнейшие работы

- **Один active writer:** текущий простой вариант для стенда.
- **Redis atomic reserve:** общий state для replicas, но Redis входит в hot path.
- **Reservation ledger:** operation IDs и terminal results дают безопасный retry,
  но требуют cleanup/reconciliation.

Выбор зависит от допустимой latency и требуемой строгости лимита.

## Связанные документы

- [Обзор проекта](../../README.md) и [первый запуск](../../GETTING_STARTED.md).
- [Карта документации](../../docs/README.md).
- [Правила разработки](../../AGENTS.md).
