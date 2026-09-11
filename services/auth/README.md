# Auth Service

Внутренний gRPC-сервис аутентификации. Проверяет email/password через Users,
выпускает JWT и валидирует токены.

## Возможности

- вход по email/password через Users Service;
- выпуск access и refresh JWT;
- проверка типа, срока и подписи токена;
- защита Login счётчиком неуспешных попыток в Redis;
- локальный rate limiter, gRPC health и OpenTelemetry instrumentation.

## Архитектура

```text
gRPC handler
    ↓
Auth service ──gRPC──> Users.ValidateCredentials
    ├── JWT service
    ├── Redis login-attempts repository
    └── metrics/tracing/logger
```

Handler отвечает за transport mapping. Service ведёт login/token flow. Проверка
пароля остаётся в Users, а Auth не читает users database напрямую.

## Структура проекта

```text
cmd/server/                  entrypoint
internal/app/                wiring и lifecycle
internal/handler/auth/       gRPC transport
internal/service/auth/       login и token flow
internal/repository/         Redis-backed login attempts
internal/config/             env configuration
pkg/mocks/                   generated mocks для тестов
```

## API

Source of truth: `shared/api/auth/v1/auth.proto`.

| RPC | Результат |
|---|---|
| `Login` | refresh token после успешной проверки credentials |
| `GetRefreshToken` | новый refresh token из валидного refresh token |
| `GetAccessToken` | access token из валидного refresh token |
| `ValidateToken` | проверка JWT из gRPC metadata |

`Login` не возвращает access token. Клиент получает его отдельным
`GetAccessToken`.

Типичный сценарий:

```text
Login(email, password) → refresh token
GetAccessToken(refresh token) → access token
ValidateToken(access token in metadata) → user_id
```

`GetRefreshToken` принимает refresh token. `ValidateToken` читает credentials из
gRPC metadata в форме, заданной protobuf/handler; при интеграции лучше использовать
generated client, а не переносить старые HTTP-примеры.

## JWT

Токены подписываются HS256. Claims текущей реализации:

```json
{
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "token_type": "access",
  "exp": 1699999999,
  "iat": 1699999000,
  "nbf": 1699999000
}
```

UUID находится в `user_id`, а не в `sub`. Refresh и access token хранятся у
клиента. `GetRefreshToken` пока только выпускает новый JWT: предыдущий refresh
token остаётся криптографически валидным до `exp`, потому что server-side
revocation/rotation state не реализован.

## Redis и rate limiting

Redis хранит счётчик неуспешных попыток для brute-force protection. Он не является
реестром активных JWT-сессий. После успешного входа счётчик сбрасывается.

Rate limiter работает в памяти одного процесса. Значение 30 requests/second по
умолчанию защищает отдельную реплику от локальной перегрузки, но не задаёт единый
кластерный лимит и само по себе не заменяет edge DDoS protection.

Если Redis недоступен, поведение Login зависит от ошибки repository и не должно
описываться как полноценная fail-open/fail-closed security policy без отдельного
теста. При нескольких Auth replicas счётчик попыток общий через Redis, а rate
limiter остаётся локальным каждой replica.

## Взаимодействие

Auth обращается к Users по gRPC `ValidateCredentials`. В Compose используется
`users:50054`.

```text
client → Auth.Login → Users.ValidateCredentials → PostgreSQL
                       ↓ success
                    refresh JWT
client → Auth.GetAccessToken(refresh JWT) → access JWT
```

В текущем Compose нет Envoy и HTTP/JSON transcoding. Примеры вызовов должны
использовать gRPC-клиент либо будущий Gateway; `curl localhost:8080/api/...` после
обычного запуска работать не будет.

## Конфигурация

Основные значения находятся в `services/auth/.env`:

```dotenv
GRPC_HOST=0.0.0.0
GRPC_PORT=50050
USER_SERVER_GRPC_HOST=users
USER_SERVER_GRPC_PORT=50054
REFRESH_TOKEN_TTL=1h
ACCESS_TOKEN_TTL=15m
CACHE_HOST=redis
INTERNAL_CACHE_PORT=6379
SECURITY_MAX_LOGIN_ATTEMPTS=6
SECURITY_LOGIN_ATTEMPTS_WINDOW=30s
```

Secrets из development `.env` нельзя использовать в production.

Полный набор и значения по умолчанию смотрите в `internal/config` и
`services/auth/.env`; README перечисляет только параметры основного flow.

## Запуск и health

Из корня репозитория:

```bash
docker compose --profile services up --build -d users auth
docker compose logs -f auth
grpcurl -plaintext localhost:50050 grpc.health.v1.Health/Check
```

Стандартный health service сообщает состояние процесса, установленное
приложением, и не гарантирует успешный вызов Users/Redis на каждый probe.

## Observability и lifecycle

Сервис использует общий Go kit для structured logging, OpenTelemetry metrics и
traces. Graceful shutdown останавливает gRPC server и зарегистрированные
dependencies. Наличие instrumentation означает, что telemetry создаётся, но её
доставка зависит от настроенного collector.

В локальном Compose Auth слушает `50050`, Users — `50054`, Redis — по внутреннему
адресу `redis:6379`. Внешний HTTP endpoint появится только вместе с Gateway.

## Ошибки интеграции

- неверные credentials не должны раскрывать, существует ли email;
- access token нельзя использовать вместо refresh token и наоборот;
- потерянный ответ `GetRefreshToken` сейчас нельзя разрешить через server-side
  rotation state;
- сетевой доступ к внутреннему gRPC API должен ограничиваться deployment policy.

## Тесты

```bash
cd services/auth
go test ./...
```

## Ограничения текущей безопасности

- gRPC transport в текущем локальном Compose plaintext;
- rate limit локален одной реплике;
- refresh revocation отсутствует;
- общий S3/Gateway authentication flow ещё не реализован.

Эти ограничения нужно учитывать при проектировании Gateway и deployment, а не
описывать как уже обеспеченные TLS, sessions или SigV4.
