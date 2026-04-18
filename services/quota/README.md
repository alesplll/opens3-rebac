# Quota Service

gRPC-сервис управления квотами хранилища для **OpenS3**.
Написан на Rust — самый быстрый сервис в проекте.

**Порт:** `50055` | **Язык:** Rust 1.82 | **БД:** Redis (DB 1)

---

## Содержание

- [Как запустить](#как-запустить)
- [Структура файлов](#структура-файлов)
- [Как работает сервис](#как-работает-сервис)
- [Почему быстрый](#почему-быстрый)
- [Redis: свой или общий](#redis-свой-или-общий)
- [Как собирается Docker](#как-собирается-docker)
- [Ручное тестирование: 10 grpcurl запросов](#ручное-тестирование)

---

## Как запустить

### Вариант 1: только этот сервис (для разработки)

Тебе нужен только Redis. Запускаем инфру и сервис:

```bash
# 1. Поднять Redis (из корня репо)
docker compose up redis -d

# 2. Перейти в директорию сервиса
cd services/quota

# 3. Убедиться что .env настроен (там уже всё правильно по умолчанию)
cat .env

# 4. Запустить
cargo run --release -p quota-service
```

Сервис стартует за ~100ms и пишет в консоль:
```
INFO quota service starting quota port=50055 env=development
INFO connected to Redis url=redis://localhost:6379/1
INFO loaded quota data from Redis usage_count=0 limits_count=0
INFO gRPC server listening addr=0.0.0.0:50055
```

### Вариант 2: весь проект (docker compose)

```bash
# Из корня репо — поднимает всё: Redis, Neo4j, Kafka, все сервисы
make up-services

# Только инфра + quota
docker compose up redis quota -d

# С observability (Jaeger, Prometheus, Grafana, Kibana)
make up-all
```

### Вариант 3: включить OTLP трейсинг (нужен observability stack)

```bash
make up-all   # поднимает сервисы + Jaeger + Prometheus + Grafana + OTel Collector
```

При запуске через docker compose `ENABLE_OTLP=true` выставляется автоматически через `docker-compose.yml`.
Для локального `cargo run` выставь вручную в `.env`:
```env
ENABLE_OTLP=true
OTLP_ENDPOINT=http://localhost:4317
```

---

## Структура файлов

Сервис разбит на **6 слоёв**. Каждый слой знает только о слое ниже себя.

```
services/quota/
├── Cargo.toml          ← зависимости Rust (tonic, dashmap, fred, ...)
├── build.rs            ← компилирует quota.proto в Rust код при сборке
├── Dockerfile          ← двухстадийная сборка: builder + runtime
├── .env                ← конфигурация по умолчанию
└── src/
    ├── main.rs         ← точка входа
    ├── app.rs          ← главный оркестратор
    ├── config.rs       ← конфигурация из env vars
    ├── metrics.rs      ← QuotaMetrics: счётчики Redis flush
    ├── domain/         ← доменные типы (бизнес-модели)
    │   ├── mod.rs
    │   ├── quota.rs    ← UsageEntry, QuotaEntry, ResourceDelta, CheckResult
    │   └── error.rs    ← QuotaError
    ├── repository/     ← слой данных (Redis)
    │   ├── mod.rs
    │   ├── traits.rs   ← интерфейс хранилища (trait)
    │   └── redis.rs    ← реализация через Redis
    ├── cache/          ← горячий in-memory кэш
    │   ├── mod.rs
    │   └── memory.rs   ← DashMap, атомарные операции
    ├── service/        ← бизнес-логика
    │   ├── mod.rs
    │   └── quota.rs    ← QuotaService
    └── transport/      ← gRPC обработчик
        ├── mod.rs
        └── grpc.rs     ← GrpcHandler, proto↔domain конвертации
```

### Что в каждом файле

#### `main.rs`
Точка входа. Три строки: загрузить `.env` файл (если есть), вызвать `app::run()`. Никакой логики.

#### `app.rs`
Главный оркестратор. Запускается в такой последовательности:
1. Читает конфиг
2. Инициализирует логгер и OTel метрики (не-фатально — при недоступном коллекторе продолжает работу)
3. Создаёт `GrpcMetrics` (Tower middleware) и `QuotaMetrics` (Redis flush)
4. Подключается к Redis
5. Загружает все данные из Redis в память (чтобы сервис не ходил в Redis при каждом запросе)
6. Запускает фоновую задачу: каждые 500ms сбрасывает изменения обратно в Redis
7. Запускает gRPC сервер с `MetricsLayer`, health check и reflection
8. Ждёт SIGTERM/SIGINT → красиво завершает все соединения

#### `config.rs`
Читает переменные окружения **один раз** при старте через `OnceLock` (Rust аналог Go `sync.Once`). После инициализации — доступен из любого места через `config::get()` без блокировок.

#### `domain/quota.rs`
Чистые типы данных — никаких зависимостей, никакого I/O:
- `UsageEntry` — сколько уже использовано (bytes, objects, buckets)
- `QuotaEntry` — лимиты (-1 = без ограничений)
- `ResourceDelta` — изменение ресурса (может быть отрицательным при удалении)
- `CheckResult` — результат проверки: `Allowed` или `Denied(DenyReason)`
- `DenyReason` — конкретная причина отказа с контекстом (сколько использовано / какой лимит)

#### `domain/error.rs`
Перечень всех ошибок сервиса через `thiserror`. Каждая ошибка автоматически превращается в gRPC Status в transport слое.

#### `repository/traits.rs`
**Интерфейс** хранилища — абстрактный контракт (Rust trait). Описывает что хранилище умеет: загрузить всё, сохранить батч, удалить субъекта, проверить здоровье. Конкретная реализация (Redis) — в отдельном файле. Благодаря этому тесты могут подменить Redis на фейковое хранилище.

#### `repository/redis.rs`
Реализация хранилища через Redis. Использует библиотеку `fred` с пулом соединений (8 штук). Схема ключей в Redis:
```
quota:usage:{subject_id}  →  HASH { bytes, objects, buckets }
quota:limit:{subject_id}  →  HASH { bytes_limit, objects_limit, buckets_limit }
```
При старте — загружает все данные через `SCAN quota:usage:*` (не `KEYS` — он блокирует Redis). При flush — пишет пачками через `HSET`.

#### `cache/memory.rs`
**Горячий путь** — самый важный файл для производительности. Хранит все данные в `DashMap` — это concurrent HashMap с мелкой блокировкой (256 шардов). 

Главная операция `check_and_reserve` работает атомарно на уровне одной записи:
```
lock shard → прочитать текущее использование →
  проверить лимит → если OK: обновить → unlock shard
                 → если превышен: не трогать → unlock shard
```
Это **без TOCTOU-гонок** (time-of-check-time-of-use race condition) и без внешнего Mutex.

#### `service/quota.rs`
Бизнес-логика. Принимает вызовы от gRPC слоя и координирует cache + repository:
- `check_quota` — проверяет user, потом bucket. Если bucket отказал — **откатывает** user резервацию
- `update_usage` — fire-and-forget обновление после успешной операции
- `set_quota` — write-through: сразу пишет в cache И в Redis (лимиты важны)
- `load_from_storage` — при старте, заполняет cache из Redis
- `flush_to_storage` — вызывается фоном каждые 500ms, записывает метрики flush (count, duration, errors) через `QuotaMetrics`

#### `metrics.rs`
Доменные метрики сервиса — отдельные от инфраструктурного `rust-kit`. `QuotaMetrics` инициализируется один раз при старте и передаётся в `QuotaService`. Содержит 4 инструмента: `redis_flush_total`, `redis_flush_errors_total`, `redis_flush_entries`, `redis_flush_duration_seconds`. Видны в Grafana в секции "Redis Persistence".

#### `transport/grpc.rs`
gRPC обработчик. Принимает protobuf сообщения, конвертирует в доменные типы, вызывает service слой, конвертирует результат обратно в protobuf. Содержит встроенный дескриптор proto файла для gRPC reflection (чтобы работал `grpcurl list`).

---

## Как работает сервис

```
При старте:
  Redis → load_all_usage/limits → MemoryCache (DashMap)

Каждый CheckQuota запрос (~200ns):
  gRPC → GrpcHandler → QuotaService → MemoryCache → ответ
  (Redis не трогается!)

Каждые 500ms (фоновая задача):
  MemoryCache.snapshot() → Redis.flush()

При SetQuota (редко):
  MemoryCache.set_limit() + Redis.flush_limits() (сразу)
```

Сервис работает **целиком в памяти** на горячем пути. Redis используется только для:
1. Загрузки данных при старте (восстановление после перезапуска)
2. Периодического сохранения (дубликат в памяти → диск)

---

## Почему быстрый

| Причина | Детали |
|---------|--------|
| **Rust** | Нет GC-пауз, нет виртуальной машины. Компилируется в машинный код |
| **Tokio** | Асинхронный рантайм. Один поток обслуживает тысячи соединений без блокировок |
| **DashMap** | Lock-free чтение для большинства операций. 256 независимых шардов → минимальная конкуренция между потоками |
| **Нет I/O на горячем пути** | CheckQuota не ходит в Redis — только DashMap в памяти |
| **Статичный бинарь** | Нет динамической линковки → быстрый старт контейнера (~50ms vs ~2s у Python) |
| **Атомарный check-and-reserve** | Нет двух отдельных операций чтения+записи → нет гонок → нет retry loops |

---

## Redis: свой или общий

**Общий Redis, но разный DB.**

В проекте один Redis контейнер. Сервисы разделяют его по database index:
- `DB 0` — authz кэш (`auth_decision:*`)
- `DB 1` — quota (`quota:usage:*`, `quota:limit:*`) ← наш

В `.env` прописано `REDIS_DB=1`. Неймспейсы не пересекаются, никаких коллизий.

**Важно для production:** В dev Redis запущен без AOF (`--appendonly no`), то есть при перезапуске Redis данные квот потеряются. Сервис просто начнёт с нулей. Для production нужно включить AOF на Redis или добавить отдельный `redis-quota` с `--appendonly yes`.

---

## Как собирается Docker

```dockerfile
FROM rust:1.82-alpine AS builder   # Alpine = musl libc = статичная линковка
  apk add musl-dev protobuf-dev    # protoc нужен для компиляции .proto
  cargo build --release            # компилируем со всеми оптимизациями

FROM alpine:3.21                   # финальный образ
  COPY server ./                   # только бинарь (~8MB)
```

**Почему образ маленький:**
- Статичный бинарь (musl) — не нужен libc в runtime образе
- Rust компилируется заранее → в финальный образ идёт только исполняемый файл
- Alpine как base — ~5MB сам по себе
- Итого образ: ~13-15MB (vs ~200MB у Python, ~50MB у Go)

**Почему слои кэшируются:**
```dockerfile
COPY Cargo.toml Cargo.lock* ./      ← копируем только манифесты
# (тут Docker закэшировал бы cargo fetch, но он внутри cargo build)
COPY shared/pkg/rust-kit ...        ← код зависимостей
COPY services/quota ...             ← наш код
RUN cargo build --release           ← компиляция
```
Если изменился только `src/`, пересобирается только наш крейт (~30s). Если не менялись зависимости — `rust-kit` тоже не пересобирается.

---

## Ручное тестирование

Убедись что сервис запущен на `:50055`.

```bash
# Проверить доступность и посмотреть все методы
grpcurl -plaintext localhost:50055 list
grpcurl -plaintext localhost:50055 list opens3.quota.v1.QuotaService
```

### Сценарий: полный цикл жизни объекта

```bash
# 1. Установить лимиты для пользователя (10 GiB, 100 бакетов, объекты не ограничены)
grpcurl -plaintext -d '{
  "subject_id": "user:alice-uuid-001",
  "bytes_limit": 10737418240,
  "objects_limit": -1,
  "buckets_limit": 100
}' localhost:50055 opens3.quota.v1.QuotaService/SetQuota

# 2. Проверить что лимиты записались
grpcurl -plaintext -d '{
  "subject_id": "user:alice-uuid-001"
}' localhost:50055 opens3.quota.v1.QuotaService/GetQuota

# 3. CreateBucket: проверить квоту (+1 бакет)
grpcurl -plaintext -d '{
  "subject_id": "user:alice-uuid-001",
  "bucket_id": "",
  "delta": {"bytes": 0, "objects": 0, "buckets": 1}
}' localhost:50055 opens3.quota.v1.QuotaService/CheckQuota
# ожидаем: allowed=true

# 4. CreateBucket прошёл — обновить потребление
grpcurl -plaintext -d '{
  "subject_id": "user:alice-uuid-001",
  "bucket_id": "",
  "delta": {"bytes": 0, "objects": 0, "buckets": 1}
}' localhost:50055 opens3.quota.v1.QuotaService/UpdateUsage

# 5. PutObject 50MB: проверить квоту (user + bucket)
grpcurl -plaintext -d '{
  "subject_id": "user:alice-uuid-001",
  "bucket_id":  "bucket:my-photos",
  "delta": {"bytes": 52428800, "objects": 1, "buckets": 0}
}' localhost:50055 opens3.quota.v1.QuotaService/CheckQuota
# ожидаем: allowed=true

# 6. PutObject прошёл — обновить потребление
grpcurl -plaintext -d '{
  "subject_id": "user:alice-uuid-001",
  "bucket_id":  "bucket:my-photos",
  "delta": {"bytes": 52428800, "objects": 1, "buckets": 0}
}' localhost:50055 opens3.quota.v1.QuotaService/UpdateUsage

# 7. Посмотреть текущее потребление пользователя
grpcurl -plaintext -d '{
  "subject_id": "user:alice-uuid-001"
}' localhost:50055 opens3.quota.v1.QuotaService/GetUsage
# ожидаем: bytes=52428800, objects=1, buckets=1

# 8. Попробовать загрузить файл который превышает лимит (установим маленький лимит)
grpcurl -plaintext -d '{
  "subject_id": "user:alice-uuid-001",
  "bytes_limit": 104857600,
  "objects_limit": -1,
  "buckets_limit": 100
}' localhost:50055 opens3.quota.v1.QuotaService/SetQuota
# лимит теперь 100MB, уже использовано 50MB

grpcurl -plaintext -d '{
  "subject_id": "user:alice-uuid-001",
  "bucket_id":  "bucket:my-photos",
  "delta": {"bytes": 104857600, "objects": 1, "buckets": 0}
}' localhost:50055 opens3.quota.v1.QuotaService/CheckQuota
# ожидаем: allowed=false, code=DENY_CODE_USER_STORAGE_EXCEEDED
# reason: "user storage exceeded: 157286400/104857600 bytes"

# 9. DeleteObject: освободить квоту
grpcurl -plaintext -d '{
  "subject_id": "user:alice-uuid-001",
  "bucket_id":  "bucket:my-photos",
  "delta": {"bytes": -52428800, "objects": -1, "buckets": 0}
}' localhost:50055 opens3.quota.v1.QuotaService/UpdateUsage

# 10. Проверить HealthCheck
grpcurl -plaintext -d '{"service": "QuotaService"}' \
  localhost:50055 opens3.quota.v1.QuotaService/HealthCheck
# ожидаем: status=SERVING (Redis доступен)
# если Redis упал: status=NOT_SERVING
```

### Дополнительно: тест лимита бакетов

```bash
# Установить лимит 2 бакета
grpcurl -plaintext -d '{
  "subject_id": "user:alice-uuid-001",
  "bytes_limit": -1,
  "objects_limit": -1,
  "buckets_limit": 2
}' localhost:50055 opens3.quota.v1.QuotaService/SetQuota

# Уже есть 1 бакет — создать ещё один (должно пройти)
grpcurl -plaintext -d '{
  "subject_id": "user:alice-uuid-001",
  "delta": {"bytes": 0, "objects": 0, "buckets": 1}
}' localhost:50055 opens3.quota.v1.QuotaService/CheckQuota
# allowed=true

# Обновить (+1 бакет, теперь 2)
grpcurl -plaintext -d '{
  "subject_id": "user:alice-uuid-001",
  "delta": {"bytes": 0, "objects": 0, "buckets": 1}
}' localhost:50055 opens3.quota.v1.QuotaService/UpdateUsage

# Попробовать создать третий бакет — должно быть отказано
grpcurl -plaintext -d '{
  "subject_id": "user:alice-uuid-001",
  "delta": {"bytes": 0, "objects": 0, "buckets": 1}
}' localhost:50055 opens3.quota.v1.QuotaService/CheckQuota
# allowed=false, code=DENY_CODE_USER_BUCKET_LIMIT_REACHED
```

---

## Что поднимается при `make up-services`

| Сервис | Порт | Зависит от |
|--------|------|-----------|
| postgres-users | 5432 | — |
| postgres-metadata | 5433 | — |
| redis | 6379 | — |
| neo4j | 7474/7687 | — |
| zookeeper | 2181 | — |
| kafka | 9092 | zookeeper |
| users | 50054 | postgres-users |
| auth | 50050 | redis, users |
| authz | 50051 | neo4j, redis, kafka |
| metadata | 50052 | — |
| storage | 50053 | — |
| **quota** | **50055** | **redis** |

Quota зависит только от Redis — самая простая зависимость в проекте.

