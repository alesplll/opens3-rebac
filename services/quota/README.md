# Quota Service

Rust 1.95 gRPC-сервис учёта лимитов пользователей и buckets. Горячее состояние
находится в памяти (`DashMap`), Redis используется для периодического сохранения.

## API и семантика

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

## Границы консистентности

Операция над одной записью DashMap синхронизирована внутри одного процесса.
Резерв пользователя и bucket — два шага с компенсацией user при отказе bucket.
Это не распределённая транзакция. Несколько активных Quota replicas держат
независимое состояние и не обеспечивают единый строгий лимит, поэтому текущий
deployment должен иметь одного active writer.

`DashMap::new()` сам выбирает число shards; это не фиксированные 256 shards и не
lock-free структура. Оценки nanoseconds для map operation нельзя переносить на
полный gRPC request без benchmark.

## Persistence

Фоновый task вызывает flush с интервалом `REDIS_FLUSH_INTERVAL_MS`. В текущем
development `.env` это 500 ms, default кода — 1000 ms. Интервал не является
гарантированной верхней границей потери: между изменением памяти, flush и
persistence Redis остаются окна сбоя.

Development Redis в Compose запущен без AOF. После перезапуска Redis persisted
quota data может исчезнуть. Включение AOF уменьшает одно из окон, но не делает
весь протокол синхронно durable.

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

## Запуск и тесты

```bash
cargo test -p quota-service
cargo run -p quota-service
```

В Docker из корня:

```bash
docker compose --profile services up --build -d quota
docker compose logs -f quota
```

`DeleteSubject` уже подключён в proto и gRPC handler. Одновременный delete и ранее
начатый flush требуют координации: удаление dirty mark само по себе не гарантирует,
что старый snapshot никогда не будет записан позже.
