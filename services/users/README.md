# Users Service

Внутренний gRPC-сервис учётных записей. Хранит пользователей и password hashes в
PostgreSQL; Auth вызывает `ValidateCredentials` при входе.

## Назначение и возможности

- CRUD учётной записи;
- отдельное изменение пароля;
- проверка email/password для Auth;
- PostgreSQL repository и транзакции;
- gRPC health, logging и OpenTelemetry instrumentation.

```text
gRPC handler → User service → repository → PostgreSQL
                    └──────→ transaction manager
Auth Service ──gRPC ValidateCredentials──┘
```

Service проверяет входные данные и управляет транзакционным сценарием. Repository
скрывает SQL, handler переводит domain errors в gRPC status.

## Архитектура и зависимости

Auth синхронно вызывает `ValidateCredentials`. Users не зависит от Auth и не
валидирует JWT самостоятельно. Проектируемое удаление связей AuthZ, metadata и
quota после удаления пользователя требует отдельного durable workflow; Kafka
producer для `user.created`/`user.deleted` сейчас отсутствует.

## Структура проекта

```text
cmd/server/                 entrypoint
internal/app/               сборка gRPC server и зависимостей
internal/handler/user/      transport layer
internal/service/user/      validation и business logic
internal/repository/user/   PostgreSQL access
internal/config/            env configuration
pkg/mocks/                  generated test mocks
migrations/                 database migrations
```

## API

Source of truth: `shared/api/user/v1/user.proto`.

| RPC | Назначение |
|---|---|
| `Create` | создать пользователя, вернуть UUID |
| `Get` | получить пользователя по UUID |
| `Update` | изменить имя и/или email |
| `UpdatePassword` | изменить пароль и записать audit history в транзакции |
| `Delete` | удалить пользователя |
| `ValidateCredentials` | проверить email/password для Auth |

`Update` не меняет пароль: для этого есть отдельный `UpdatePassword`.

`Create` возвращает UUID пользователя и не выпускает JWT. `UpdatePassword`
записывает изменение через предусмотренный транзакционный flow; детали таблиц и
ограничений задаются миграциями, а не README.

## Данные и основные сценарии

### Граница доверия

Текущая реализация предназначена для внутренней сети. gRPC server не извлекает
JWT identity и не проверяет, что `user_id` принадлежит вызывающему. TLS credentials
на server также не настроены. Поэтому self-only policy должна обеспечиваться
будущим Gateway/межсервисной аутентификацией либо отдельным изменением Users.

Сервис сейчас не публикует `user.created` и `user.deleted` в Kafka. Cascade после
удаления пользователя остаётся проектируемым flow.

## Конфигурация

### Полный перечень переменных Go-конфигурации

Default ниже относится к коду, а не к development `.env`. Для группы PG можно
задать `PG_DSN` либо полный набор host/port/database/user/password.

| Переменная | Default / требование | Раздел конфигурации |
|---|---|---|
| `GRPC_HOST` | задать явно | [grpc.go](internal/config/env/grpc.go) |
| `GRPC_PORT` | задать явно | [grpc.go](internal/config/env/grpc.go) |
| `LOGGER_LEVEL` | задать явно | [logger.go](internal/config/env/logger.go) |
| `LOGGER_AS_JSON` | задать явно | [logger.go](internal/config/env/logger.go) |
| `LOGGER_ENABLE_OLTP` | задать явно | [logger.go](internal/config/env/logger.go) |
| `OTEL_SERVICE_NAME` | задать явно | [logger.go](internal/config/env/logger.go) |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | задать явно | [logger.go](internal/config/env/logger.go) |
| `OTEL_ENVIRONMENT` | задать явно | [logger.go](internal/config/env/logger.go) |
| `OTEL_SERVICE_VERSION` | задать явно | [metrics.go](internal/config/env/metrics.go) |
| `OTEL_METRICS_PUSH_TIMEOUT` | задать явно | [metrics.go](internal/config/env/metrics.go) |
| `PG_DSN` | не задан | [pg.go](internal/config/env/pg.go) |
| `PG_HOST` | не задан | [pg.go](internal/config/env/pg.go) |
| `PG_PORT_INNER` | не задан | [pg.go](internal/config/env/pg.go) |
| `PG_DATABASE_NAME` | не задан | [pg.go](internal/config/env/pg.go) |
| `PG_USER` | не задан | [pg.go](internal/config/env/pg.go) |
| `PG_PASSWORD` | не задан | [pg.go](internal/config/env/pg.go) |
| `PG_SSL_MODE` | `disable` | [pg.go](internal/config/env/pg.go) |
| `PG_TIMEOUT` | `5s` | [pg.go](internal/config/env/pg.go) |
| `PG_LOGGING` | `false` | [pg.go](internal/config/env/pg.go) |
| `RATE_LIMITER_LIMIT` | `100` | [rate_limiter.go](internal/config/env/rate_limiter.go) |
| `RATE_LIMITER_PERIOD` | `1s` | [rate_limiter.go](internal/config/env/rate_limiter.go) |

Compose использует `services/users/.env`. Основные значения текущего локального
стенда:

```dotenv
GRPC_HOST=0.0.0.0
GRPC_PORT=50054
PG_HOST=postgres-users
PG_PORT_INNER=5432
PG_DATABASE_NAME=users_db
PG_USER=users_user
PG_PASSWORD=users_password
OTEL_EXPORTER_OTLP_ENDPOINT=otel-collector:4317
```

Не копируйте эти development credentials в production.

Полный перечень env и parsing rules находится в `internal/config`. При локальном
запуске вне Compose адрес PostgreSQL нужно заменить с container DNS на доступный
host address.

## Запуск

### Локальный Go и зависимости в Docker

Из корня репозитория:

```bash
docker compose --profile services up --build -d migrator-users
docker compose --profile services stop users
cd services/users
PG_HOST=localhost PG_PORT_INNER=5432 OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317 go run ./cmd/server
```

Go 1.24.1 читает сервисный `.env` относительно текущей папки; переменные shell
переопределяют значения файла. Для Users дождитесь завершения мигратора с кодом 0.
Экспорт телеметрии требует `make up-observability` в отдельном терминале.

### Docker

Из корня репозитория:

```bash
docker compose --profile services up --build -d users
docker compose logs -f users
```

Полный локальный профиль поднимает `migrator-users` перед сервисом.

## Примеры использования

Команды для development-стенда с установленным `grpcurl`:

```bash
grpcurl -plaintext -d '{"user_info":{"name":"Docs Example","email":"docs@example.com"},"password":"Docs-example-123!","password_confirm":"Docs-example-123!"}' \
  localhost:50054 user_v1.UserV1/Create
```

Скопируйте `id` из ответа и подставьте вместо `<user-id>`:

```bash
grpcurl -plaintext -d '{"id":"<user-id>"}' localhost:50054 user_v1.UserV1/Get
grpcurl -plaintext -d '{"user_id":"<user-id>","name":"Updated Example"}' \
  localhost:50054 user_v1.UserV1/Update
```

`Get` принимает `id`, а Update/Delete/UpdatePassword — `user_id`. Поля
`name`/`email` в Update — protobuf wrappers: отсутствие означает «не изменять».
Повторный Create с тем же email не является идемпотентным созданием.
После проверки тестовую запись можно удалить:

```bash
grpcurl -plaintext -d '{"user_id":"<user-id>"}' localhost:50054 user_v1.UserV1/Delete
```

## Наблюдаемость и диагностика

### Health

```bash
grpcurl -plaintext localhost:50054 grpc.health.v1.Health/Check
```

Стандартный health status показывает состояние процесса, установленное при
старте. Он не выполняет PostgreSQL query на каждый вызов.

### Observability и отказоустойчивость

Handler/service/repository создают telemetry через общий Go kit. Экспорт требует
доступного OTEL Collector. Graceful shutdown завершает gRPC server и соединения.

Локальный rate limiting, если включён в server middleware, относится к одной
replica. Он не является глобальным лимитом и не заменяет защиту внешнего Gateway.
PostgreSQL остаётся источником истины; гарантии backup/failover определяются его
deployment, а не кодом Users.

## Разработка и тесты

```bash
cd services/users
go test ./...
```

Unit tests расположены рядом с handler/service packages и используют generated
mocks из `pkg/mocks`. Текущий набор не является утверждением о полном e2e или
security coverage.

При изменении методов рекомендуется отдельно проверять validation failure,
repository errors, transaction rollback и отсутствие нежелательных вызовов
repository. Интеграционные тесты с PostgreSQL относятся к отдельному e2e профилю.

## Ограничения и дальнейшие работы

JWT/TLS identity, self-only authorization и Kafka lifecycle events не реализованы.
Проверки формата email/password не заменяют аутентификацию вызывающего RPC.

## Связанные документы

- [Обзор проекта](../../README.md) и [первый запуск](../../GETTING_STARTED.md).
- [Карта документации](../../docs/README.md).
- [Правила разработки](../../AGENTS.md).
