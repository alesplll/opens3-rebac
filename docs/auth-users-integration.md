# Интеграция Go-сервисов: auth + users

> Ветка: `feat/auth_users/service-integration`

Эта ветка переводит сервисы **auth** и **users** из изолированных проектов в единое монорепо.
Если ты подключаешься к проекту или делаешь review — читай этот документ.

---

## Что изменилось

### Go Workspace

В корне репо появился `go.work`. Все Go-модули теперь объединены в один workspace:

```
go.work
  └── shared/              ← общие контракты и инфраструктура
  └── services/auth/       ← auth сервис
  └── services/users/      ← users сервис
```

Для работы с кодом:
```bash
# Один раз после clone
go work sync

# Сборка всех модулей
go build ./services/auth/...
go build ./services/users/...
go build ./shared/...

# Тесты
go test ./services/users/...
```

---

### Структура shared/

```
shared/
├── go.mod
├── api/                        ← proto-файлы (источник истины)
│   ├── user/v1/user.proto
│   ├── auth/v1/auth.proto
│   ├── authz/v1/authz.proto
│   ├── metadata/v1/metadata.proto
│   └── storage/v1/storage.proto
└── pkg/
    ├── user/v1/                ← сгенерированный Go-код (user gRPC)
    ├── auth/v1/                ← сгенерированный Go-код (auth gRPC)
    └── kit/                    ← общая инфраструктура для Go-сервисов
        ├── tokens/             ← JWT: UserClaims, TokenService
        ├── tokens/jwt/         ← JWT реализация
        ├── contextx/claimsctx/ ← user_id / email в контексте
        ├── contextx/ipctx/     ← IP в контексте
        ├── client/db/          ← PostgreSQL клиент
        ├── client/db/pg/
        ├── client/db/transaction/
        ├── closer/             ← graceful shutdown
        ├── logger/             ← zap logger
        ├── metric/             ← OTEL метрики
        ├── middleware/metrics/      ← gRPC interceptor метрик
        ├── middleware/ratelimiter/  ← gRPC interceptor rate limit
        ├── middleware/validation/   ← gRPC interceptor валидации
        ├── circuitbreaker/     ← circuit breaker (для Gateway)
        ├── ratelimiter/        ← rate limiter (для Gateway)
        ├── kafka/              ← Kafka producer / consumer
        ├── sys/                ← domain errors
        └── tracing/            ← OpenTelemetry tracing
```

**Импорты в сервисах:**
```go
// proto-контракты
"github.com/alesplll/opens3-rebac/shared/pkg/go/user/v1"
"github.com/alesplll/opens3-rebac/shared/pkg/go/auth/v1"

// инфраструктура
"github.com/alesplll/opens3-rebac/shared/pkg/kit/tokens"
"github.com/alesplll/opens3-rebac/shared/pkg/kit/logger"
"github.com/alesplll/opens3-rebac/shared/pkg/kit/kafka/producer"
```

---

### Новые имена модулей

| Сервис | Было | Стало |
|--------|------|-------|
| auth   | `github.com/WithSoull/AuthService` | `github.com/alesplll/opens3-rebac/services/auth` |
| users  | `github.com/WithSoull/UserServer` | `github.com/alesplll/opens3-rebac/services/users` |
| shared | — | `github.com/alesplll/opens3-rebac/shared` |

---

### Что изменилось в users

**UUID вместо serial ID:**
- Таблица `users.id` переведена с `serial` на `uuid` (миграция `20260309000000_uuid_migration.sql`)
- Все методы репозитория, сервиса и proto принимают/возвращают `string` вместо `int64`

**Убрана Kafka:**
- Сервис больше не публикует события `UserCreated` / `UserDeleted`
- Зависимость `sarama` удалена из `go.mod`

**Убрана JWT-верификация:**
- Users больше не проверяет токены самостоятельно
- `user_id` теперь передаётся явно в теле запроса для `Delete`, `Update`, `UpdatePassword`
- Логика: Gateway аутентифицирует через auth → передаёт `user_id` в downstream сервисы

**Порт изменён:** `50052` → `50054` (освобождаем `50052` для metadata)

### Что изменилось в auth

- Модуль переименован
- JWT теперь хранит `user_id` как `string` (UUID) — раньше был `int64(0)` как заглушка
- Зависимость на `platform_common` заменена на `shared/kit`

---

### Proto: DeleteRequest

Метод `UserV1.Delete` теперь принимает `DeleteRequest` вместо `google.protobuf.Empty`:

```protobuf
message DeleteRequest {
  string user_id = 1;
}

rpc Delete(DeleteRequest) returns (google.protobuf.Empty);
```

`Update` и `UpdatePassword` также получили поле `user_id = 1`.

---

### Порты сервисов

| Сервис | Порт |
|--------|------|
| auth   | `50050` (gRPC) |
| users  | `50054` (gRPC) |
| authz  | `50051` (gRPC) |
| metadata | `50052` (placeholder) |
| data-node | `50053` (placeholder) |
| gateway | `8080` (HTTP, placeholder) |

---

## Запуск

Единая точка входа — `docker-compose.yml` в корне репо.

```bash
# Только инфраструктура (postgres, redis, neo4j, kafka)
docker compose up

# Инфраструктура + сервисы
docker compose --profile services up

# Инфраструктура + мониторинг (jaeger, prometheus, grafana)
docker compose --profile observability up

# Всё
docker compose --profile services --profile observability up
```

**Kafka внутри Docker:** контейнеры подключаются через `kafka:29092` (не `9092`).
Порт `9092` открыт наружу для инструментов на хост-машине.

---

## Для Gateway (Макс)

В `shared/pkg/kit/` уже есть:
- `circuitbreaker/` — Circuit Breaker middleware для gRPC клиентов
- `ratelimiter/` — Rate Limiter
- `kafka/producer/` и `kafka/consumer/` — Kafka клиенты

Добавь `./services/gateway` в `go.work` когда будешь готов.

---

## Известные ограничения

- Тесты только в `services/users` (handler + service layer)
- Мониторинг (observability profile) не тестировался — конфиги перенесены из `platform_common`
- Authz, metadata, data-node — их docker-compose не трогали
