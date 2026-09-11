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
