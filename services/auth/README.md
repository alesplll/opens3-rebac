# AuthService

Микросервис аутентификации пользователей по токенам

## Функциональность
- **Login** - аутентификация пользователя с выдачей refresh токена
- **GetRefreshToken** - обновление refresh токена
- **GetAccessToken** - получение access токена по refresh токену
- **ValidateToken** - валидация JWT токена для защищённых эндпоинтов

## Технологический стек
- **Go:** 1.24+
- **gRPC** - RPC API между сервисами
- **Redis** - brute force protection
- **OpenTelemetry (OTEL):**
  - **Metrics:** Prometheus → Grafana
  - **Logging:** uber-go/zap → Elasticsearch → Kibana
  - **Tracing:** Jaeger
- **JWT:** github.com/golang-jwt/jwt (алгоритм HS256)

## Архитектура

Сервис построен на основе **трёхслойной архитектуры**:

```
┌─────────────────────────────────────┐
│         Handler Layer               │  ← gRPC handlers
├─────────────────────────────────────┤
│         Service Layer               │  ← Бизнес-логика (JWT generation, validation)
├─────────────────────────────────────┤
│       Repository Layer              │  ← Работа с Redis (brute force protection)
└─────────────────────────────────────┘
```

**Ключевые архитектурные решения:**

- **Dependency Injection (DI) контейнер** - управление зависимостями между слоями
- **Graceful Shutdown** - корректное завершение работы с закрытием соединений

## Структура проекта

```
.
├── cmd/
│   └── server/
│       └── main.go                    # Entry point
├── internal/
│   ├── app/
│   │   ├── app.go                     # Application setup
│   │   └── service_provider.go        # DI container
│   ├── handler/
│   │   └── auth/                      # gRPC handlers
│   ├── service/
│   │   └── auth/                      # Бизнес-логика
│   ├── repository/
│   │   └── auth/                      # Redis repository
│   ├── client/
│   │   ├── cache/redis/               # Redis client
│   │   └── grpc/user/                 # UserServer gRPC client
│   ├── config/                        # Конфигурация из .env
│   ├── errors/domain/                 # Доменные ошибки
│   ├── model/                         # Domain models
│   ├── utils/                         # Утилиты (email validation)
│   └── validator/                     # Валидация запросов
├── api/auth/v1/
│   └── auth.proto                     # Protocol Buffers
├── pkg/auth/v1/                       # Сгенерированный Go код
├── ca.cert                            # CA сертификат
├── service.pem                        # TLS сертификат для похода в UserServer
├── docker-compose.yaml
├── Dockerfile
└── Makefile
```

## Конфигурация

### Переменные окружения

Создайте `.env` файл на основе примера:

```
# ====== gRPC ======
# Internal gRPC server
GRPC_HOST=0.0.0.0
GRPC_PORT=50050

# External gRPC (UserServer)
USER_SERVER_GRPC_HOST=user-server
USER_SERVER_GRPC_PORT=50051

# ======= JWT =======
REFRESH_TOKEN_SECRET=refresh_secret
ACCESS_TOKEN_SECRET=access_secret
REFRESH_TOKEN_TTL=1h
ACCESS_TOKEN_TTL=15m

# ====== Redis ======
CACHE_HOST=redis
EXTERNAL_CACHE_PORT=6380
INTERNAL_CACHE_PORT=6379
CACHE_CONNECTION_TIMEOUT=5s
CACHE_MAX_IDLE=10
CACHE_IDLE_TIMEOUT=300s

# Security (Brute Force Protection)
SECURITY_MAX_LOGIN_ATTEMPTS=6
SECURITY_LOGIN_ATTEMPTS_WINDOW=30s

# ====== Logger ======
LOGGER_LEVEL=debug
LOGGER_AS_JSON=false
LOGGER_ENABLE_OLTP=true

# ======= OTEL ========
OTEL_EXPORTER_OTLP_ENDPOINT=otel-collector:4317
OTEL_SERVICE_NAME=auth_service
OTEL_ENVIRONMENT=development
OTEL_SERVICE_VERSION=1.0.0
OTEL_METRICS_PUSH_TIMEOUT=1s

# ==== Rate Limiter ====
RATE_LIMITER_LIMIT=30
RATE_LIMITER_PERIOD=1s
```

## API

### gRPC Endpoints

Сервис предоставляет следующие gRPC методы (описаны в `api/auth/v1/auth.proto`):

- **Login** - аутентификация пользователя с выдачей refresh токена
- **GetRefreshToken** - обновление refresh токена (rotation)
- **GetAccessToken** - получение access токена по refresh токену
- **ValidateToken** - валидация JWT токена для других сервисов


## JWT Токены
### Типы токенов
**Refresh Token**
- **Назначение:** долгосрочная аутентификация, используется для получения access токенов
- **TTL:** 1 час (конфигурируется через `REFRESH_TOKEN_TTL`)
- **Хранение:** клиент (не хранится на сервере)
- **Алгоритм:** HS256
  
**Access Token**
- **Назначение:** краткосрочный доступ к защищённым ресурсам
- **TTL:** 15 минут (конфигурируется через `ACCESS_TOKEN_TTL`)
- **Хранение:** клиент (не хранится на сервере)
- **Алгоритм:** HS256

### JWT Claims

```
{
  "user_id": 1234,
  "email": "user@example.com",
  "exp": 1699999999,  // expiration time
  "iat": 1699999000   // issued at
}
```

## Безопасность

### Brute Force Protection

Защита от перебора паролей на уровне Redis:
- **Максимум попыток:** 6 (конфигурируется через `SECURITY_MAX_LOGIN_ATTEMPTS`)
- **Временное окно:** 30 секунд (конфигурируется через `SECURITY_LOGIN_ATTEMPTS_WINDOW`)
- **Блокировка:** временная блокировка после превышения лимита

**Реализация:**
- Счётчик попыток хранится в Redis с ключом `login_attempts:{email}`
- TTL счётчика соответствует временному окну
- После успешного входа счётчик сбрасывается

### Rate Limiting

Защита от DDoS и перегрузки:
- **Лимит:** 30 запросов/сек (конфигурируется через `RATE_LIMITER_LIMIT`)
- **Период:** 1 секунда
- **Реализация:** middleware из платформенной библиотеки

## Взаимодействие с другими сервисами

### UserServer (gRPC Client)

AuthService вызывает UserServer для валидации credentials:

**Метод:** `ValidateCredentials`

**Процесс:**
1. AuthService получает email/password от клиента
2. Устанавливает gRPC соединение с UserServer
3. Вызывает `ValidateCredentials` метод
4. Получает подтверждение или ошибку
5. Генерирует JWT токены при успехе

**Конфигурация:**
- `USER_SERVER_GRPC_HOST` - хост UserServer
- `USER_SERVER_GRPC_PORT` - порт UserServer (50051)

## Мониторинг и Observability

### OpenTelemetry (OTEL)

Интегрирован мониторинг из [платформенной библиотеки](https://github.com/WithSoull/platform_common):

**Traces**
- Распределённая трассировка между AuthService и UserServer
- Trace context propagation через gRPC metadata

**Metrics**
- RPS (requests per second)
- Latency percentiles (p50, p95, p99)
- Error rate по каждому методу

**Logs**
- Структурированное логирование (uber-go/zap)
- Dual output: stdout + OTEL Collector → Elasticsearch → Kibana

### Health Check

```
# gRPC health check
grpcurl -plaintext localhost:50050 grpc.health.v1.Health/Check

```

## Отказоустойчивость

### Rate Limiter

- **Реализация:** middleware из платформенной библиотеки
- **Защита:** 30 запросов/сек
- **Ответ при превышении:** `RESOURCE_EXHAUSTED` (HTTP 429)

### Graceful Shutdown

При получении сигнала завершения (SIGINT/SIGTERM):
1. Остановка приёма новых запросов
2. Завершение обработки текущих запросов
3. Закрытие соединений с Redis
4. Закрытие gRPC клиента к UserServer
5. Остановка gRPC сервера

## Сетевое взаимодействие

Все внешние запросы к сервису проходят через **Envoy proxy** (конфигурация в платформенной библиотеке):
- gRPC-JSON transcoding
- Load balancing
- TLS termination
- Rate limiting на уровне proxy

## Типичные сценарии использования

### 1. Полный flow аутентификации

```
# 1. Login - получение refresh токена
curl -X POST http://localhost:8080/api/v1/auth \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com", "password": "password123"}'

# Response: {"refresh_token": "eyJhbGc..."}

# 2. Получение access токена
curl -X POST http://localhost:8080/api/v1/auth/access \
  -H "Content-Type: application/json" \
  -d '{"refresh_token": "eyJhbGc..."}'

# Response: {"access_token": "eyJhbGc..."}

# 3. Использование access токена для защищённых ресурсов
curl -X GET http://localhost:8080/api/v1/protected \
  -H "Authorization: Bearer eyJhbGc..."
```

### 2. Refresh token rotation

```
# Обновление refresh токена
curl -X POST http://localhost:8080/api/v1/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{"refresh_token": "old_token"}'

# Response: {"refresh_token": "new_token"}
```

### 3. Валидация токена (для других сервисов)

```
# Проверка валидности токена
curl -X POST http://localhost:8080/api/v1/auth/validate \
  -H "Authorization: Bearer eyJhbGc..."

# Response: 200 OK или 401 Unauthorized
```

## Обработка ошибок

Кастомные доменные ошибки с маппингом в gRPC статус коды:

| Доменная ошибка | gRPC Code | HTTP Code | Описание |
|-----------------|-----------|-----------|----------|
| `INVALID_CREDENTIALS` | `Unauthenticated` | 401 | Неверный email/password |
| `TOKEN_EXPIRED` | `Unauthenticated` | 401 | Истёк срок действия токена |
| `INVALID_TOKEN` | `Unauthenticated` | 401 | Невалидная подпись токена |
| `TOO_MANY_ATTEMPTS` | `PermissionDenied` | 403 | Превышен лимит попыток входа |
| `USER_NOT_FOUND` | `NotFound` | 404 | Пользователь не найден (от UserServer) |
| `RATE_LIMIT_EXCEEDED` | `ResourceExhausted` | 429 | Превышен rate limit |

## Зависимости

### Внешние сервисы

- **UserServer** (gRPC) - валидация credentials
- **Redis** - сессии, rate limiting, brute force protection
- **OTEL Collector** - мониторинг и observability
- **Envoy Proxy** - маршрутизация

### [Платформенная библиотека](https://github.com/WithSoull/platform_common)

Переиспользуемые компоненты:
- JWT Generator/Verifier
- OTEL instrumentation (traces, metrics, logs)
- Rate Limiter middleware
- Envoy configuration
- Error handling utilities
