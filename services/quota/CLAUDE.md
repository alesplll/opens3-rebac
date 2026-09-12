# Quota context for assistants

Перед изменением прочитайте [README.md](README.md) и корневой
[AGENTS.md](../../AGENTS.md).

- `CheckQuota` — check-and-reserve, а не read-only check.
- Не вызывайте `UpdateUsage(+delta)` после allowed check с той же delta.
- При ошибке операции нужна компенсация `UpdateUsage(-delta)`.
- Текущий API не делает retries exactly-once: operation ID отсутствует.
- Consistency ограничена одним процессом; несколько active replicas небезопасны.
- `DashMap` использует fine-grained locking и сам выбирает shard count.
- Redis flush asynchronous; development Redis работает без AOF.
- `DeleteSubject` есть в proto и handler, но требует защиты от in-flight flush.
- Kafka accounting consumers пока не являются реализованным backup path.

Проверка:

```bash
cargo fmt --all -- --check
cargo clippy -p quota-service --all-targets -- -D warnings
cargo test -p quota-service
```

## Навигация и архитектура

```text
transport/grpc.rs → service/quota.rs → cache/memory.rs
                           └───────→ repository/traits.rs → repository/redis.rs
app.rs: startup load, background flush, gRPC, shutdown
config.rs: OnceLock и env; main.rs: dotenvy + tokio entrypoint
```

Domain types: `UsageEntry`, `QuotaEntry`, `ResourceDelta`, `CheckResult`,
`DenyReason` в `src/domain`. Transport переводит proto/domain и gRPC errors;
бизнес-семантика reserve/rollback остаётся в service/cache.

CheckQuota резервирует user, затем необязательный bucket; при отказе bucket
компенсирует user. GetUsage для неизвестного subject возвращает нулевой usage.
Нельзя делать вывод о межпроцессной атомарности из shard lock DashMap.
При старте состояние загружается из Redis. SetQuota сохраняет limits, а изменения
usage попадают в periodic flush. Ошибка flush должна возвращать dirty keys для
повтора без перезаписи более свежих значений памяти старым snapshot.

## Запуск и конфигурация

Полные команды и env: [README](README.md). Для local run нужен Rust toolchain,
`protoc` и Redis. Docker build использует Rust 1.95; не выводите MSRV только из
версии Docker image. `cargo run` ищет `.env` от текущей папки, поэтому запуск из
корня использует defaults/root env, а из `services/quota` — сервисный `.env`.
В Compose `REDIS_HOST`, `ENABLE_OTLP`, `OTLP_ENDPOINT` переопределены явно.

## Разработка и тесты

Команды выполняются из корня:

```bash
cargo test -p quota-service --test grpc
TEST_REDIS_URL=redis://localhost:6379/15 cargo test -p quota-service --test redis -- --include-ignored
make generate-quota-go
make generate-quota-py
```

Последние две команды требуют корневого `make install-deps` и обновляют shared
clients. Rust stubs создаёт `build.rs`; сгенерированный Tonic код не редактируется.

`tests/grpc.rs` запускает in-process server. Redis suite ignored по умолчанию и
может завершиться без проверки при недоступном Redis — проверяйте skips/logs.
Unit tests находятся рядом с реализацией. Проверяйте пользовательский результат:
reserve, bucket rollback, отрицательную compensation, fail/retry flush и
сохранение concurrent updates. Fixed nanosecond/throughput claims требуют
отдельных benchmarks.

## Диагностика

Metrics определены в `src/metrics.rs`; dashboard —
`infra/otel/grafana/dashboards/quota.json` от корня. Для сбоев persistence следите
за flush errors/entries/duration, а не только за gRPC health. Успех health не
доказывает, что последнее изменение памяти уже переживёт падение процесса.
