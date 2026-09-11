# Auth Service

Внутренний gRPC-сервис аутентификации. Проверяет email/password через Users,
выпускает JWT и валидирует токены.

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

## Запуск и health

Из корня репозитория:

```bash
docker compose --profile services up --build -d users auth
docker compose logs -f auth
grpcurl -plaintext localhost:50050 grpc.health.v1.Health/Check
```

Стандартный health service сообщает состояние процесса, установленное
приложением, и не гарантирует успешный вызов Users/Redis на каждый probe.

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