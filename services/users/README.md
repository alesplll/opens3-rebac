# Users Service

Внутренний gRPC-сервис учётных записей. Хранит пользователей и password hashes в
PostgreSQL; Auth вызывает `ValidateCredentials` при входе.

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

## Граница доверия

Текущая реализация предназначена для внутренней сети. gRPC server не извлекает
JWT identity и не проверяет, что `user_id` принадлежит вызывающему. TLS credentials
на server также не настроены. Поэтому self-only policy должна обеспечиваться
будущим Gateway/межсервисной аутентификацией либо отдельным изменением Users.

Сервис сейчас не публикует `user.created` и `user.deleted` в Kafka. Cascade после
удаления пользователя остаётся проектируемым flow.

## Структура

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

## Конфигурация

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

## Запуск

Из корня репозитория:

```bash
docker compose --profile services up --build -d users
docker compose logs -f users
```

Полный локальный профиль поднимает `migrator-users` перед сервисом.

## Health

```bash
grpcurl -plaintext localhost:50054 grpc.health.v1.Health/Check
```

Стандартный health status показывает состояние процесса, установленное при
старте. Он не выполняет PostgreSQL query на каждый вызов.

## Тесты

```bash
cd services/users
go test ./...
```

Unit tests расположены рядом с handler/service packages и используют generated
mocks из `pkg/mocks`. Текущий набор не является утверждением о полном e2e или
security coverage.
