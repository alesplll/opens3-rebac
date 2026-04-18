# CLAUDE.md — Quota Service

## Что это

gRPC-сервис управления квотами хранилища. Отвечает на вопрос:
**"есть ли у пользователя/бакета достаточно квоты для выполнения операции?"**

Написан на **Rust** (tonic + tokio + DashMap + Redis/fred).

---

## Команды

### Локальный запуск (только Redis нужен)

```bash
cd services/quota
# Поднять только Redis
docker compose up redis -d

# Запустить сервис
cargo run --release -p quota-service

# gRPC на :50055, reflection включён
grpcurl -plaintext localhost:50055 list
```

### Docker

```bash
# Только квота + инфра
docker compose --profile services up redis quota -d

# Или всё сразу
make up-services
```

### Тесты

```bash
cargo test -p quota-service
cargo test -p quota-service -- --nocapture   # с логами
```

### Proto regeneration (Go-стабы для Gateway)

```bash
make generate-quota-go   # → shared/pkg/go/quota/v1/
make generate-quota-py   # → shared/pkg/py/quota/v1/  (если понадобится)
```

---

## Архитектура

```
gRPC Request
  └─► transport/grpc.rs     (proto ↔ domain конвертация)
        └─► service/quota.rs (бизнес-логика, reserve+rollback)
              └─► cache/memory.rs     (DashMap, горячий путь ~100ns)
              └─► repository/redis.rs (persistence, flush каждые 500ms)
```

### Горячий путь CheckQuota

```
CheckQuota(subject, bucket, delta)
  1. Читаем лимит из DashMap<limits> — O(1), ~50ns, без блокировок
  2. DashMap.entry(subject).and_modify(|usage| {
         if would_exceed(usage, limit, delta) → DENY, не трогаем usage
         else → usage.apply(delta)              → ALLOW, резервируем
     })                                         ← атомарно (shard lock)
  3. Если есть bucket_id → повторяем для бакета
     При отказе бакета → rollback user reservation
  4. Возвращаем CheckQuotaResponse (~200ns total)
```

Нет `await`, нет I/O — всё в памяти. Redis пишется в фоне.

---

## Файловая структура

```
src/
  main.rs            — точка входа, dotenvy, tokio::main
  app.rs             — App::run(): инит всего, MetricsLayer, запуск gRPC
  config.rs          — OnceLock singleton, читается один раз
  metrics.rs         — QuotaMetrics: flush_total, flush_errors, flush_entries, flush_duration
  domain/
    quota.rs         — UsageEntry, QuotaEntry, ResourceDelta, CheckResult, DenyReason
    error.rs         — QuotaError (thiserror)
  repository/
    traits.rs        — trait QuotaRepository (интерфейс хранилища)
    redis.rs         — RedisRepository (fred pool, SCAN/HMGET/HSET)
  cache/
    memory.rs        — MemoryCache (DashMap, атомарный check-and-reserve)
  service/
    quota.rs         — QuotaService (бизнес-логика, reserve+rollback, flush metrics)
  transport/
    grpc.rs          — GrpcHandler (tonic impl, proto↔domain)
```

---

## Конфигурация (env vars)

| Var | Default | Описание |
|-----|---------|----------|
| `GRPC_PORT` | `50055` | gRPC порт |
| `REDIS_HOST` | `localhost` | Redis хост |
| `REDIS_PORT` | `6379` | Redis порт |
| `REDIS_DB` | `1` | Redis DB индекс (0 занят authz) |
| `REDIS_FLUSH_INTERVAL_MS` | `500` | Интервал flush в Redis |
| `DEFAULT_USER_BYTES_LIMIT` | `10737418240` | 10 GiB по умолчанию |
| `DEFAULT_USER_BUCKETS_LIMIT` | `100` | Лимит бакетов |
| `DEFAULT_USER_OBJECTS_LIMIT` | `-1` | Без ограничений |
| `ENABLE_OTLP` | `false` | OTel экспорт; в docker-compose переопределяется в `true` автоматически |
| `OTLP_ENDPOINT` | `http://otel-collector:4317` | Эндпоинт коллектора |

---

## Соглашения entity ID

Такие же как в AuthZ:
- `user:{uuid}` — пользователь
- `bucket:{name}` — бакет

---

## Маппинг S3-операций

| Операция | Gateway вызывает |
|----------|-----------------|
| PutObject | `CheckQuota(user+bucket, Δbytes=size, Δobjects=1)` → `UpdateUsage` |
| DeleteObject | `UpdateUsage(user+bucket, Δbytes=-size, Δobjects=-1)` |
| CreateBucket | `CheckQuota(user, Δbuckets=1)` → `UpdateUsage` |
| DeleteBucket | `UpdateUsage(user, Δbuckets=-1)` |
| GetObject / List / Head | не вызывается |

---

## Границы сервиса

| НЕ делает |
|-----------|
| Аутентификацию (JWT — это Gateway) |
| Авторизацию (разрешения — это AuthZ) |
| Хранение объектов |
| Знание о blob_id или ключах S3 |

---

## Observability

- **gRPC метрики** — через `rust-kit::middleware::grpc::MetricsLayer` (Tower): `grpc_quota_requests_total`, `grpc_quota_response_total`, `grpc_quota_histogram_response_time_seconds`
- **Redis метрики** — через `QuotaMetrics` в `metrics.rs`: `quota_redis_flush_total`, `quota_redis_flush_errors_total`, `quota_redis_flush_entries`, `quota_redis_flush_duration_seconds`
- **Трейсы** — `#[instrument]` на всех методах repository, видны в Jaeger
- **Grafana дашборд** — `infra/otel/grafana/dashboards/quota.json`, 5 секций: Summary / Traffic / Latency / Errors / Redis Persistence

---

## Известные ограничения / TODO

- `flush_to_storage` пишет весь DashMap каждые 500ms — для >100k пользователей
  стоит добавить dirty-set чтобы писать только изменённые записи
- Redis используется без AOF в dev (`--appendonly no` в docker-compose);
  в prod нужно включить `--appendonly yes` на redis-quota
- SetQuota write-through синхронный — при большом количестве вызовов можно
  сделать batch с debounce
- `delete_subject` в cache/repository/service помечен `#[allow(dead_code)]` — Phase 2, пока не подключён к gRPC
