# UserServer
Микросервис управления пользователями. Отвечает за создание, обновление, удаление пользователей и валидацию их учетных данных для auth.

## Функциональность
- **Создание пользователя** - регистрация новых пользователей в системе
- **Просмотр информации о пользователе** - получение профиля пользователя
- **Удаление пользователя** - удаление собственной учетной записи (требует авторизации)
- **Обновление пароля** - изменение пароля пользователя
- **Валидация учетных данных** - проверка credentials для Auth Service

## Технологический стек
- **Go:** 1.24.1
- **База данных:** PostgreSQL 16
- **RPC:** gRPC + gRPC-Gateway (HTTP/JSON → gRPC mapping)
- **Message Broker:** Kafka
- **Мониторинг:** OpenTelemetry (OTEL)

## Архитектура
Сервис построен на основе **трёхслойной архитектуры**:
```
┌─────────────────────────────────────┐
│         Handler Layer               │  ← gRPC handlers + HTTP (gRPC-Gateway)
├─────────────────────────────────────┤
│         Service Layer               │  ← Бизнес-логика
├─────────────────────────────────────┤
│        Repository Layer             │  ← Работа с PostgreSQL
└─────────────────────────────────────┘
```
**Ключевые архитектурные решения:**
- **Dependency Injection (DI) контейнер** - управление зависимостями между слоями
- **Graceful Shutdown** - корректное завершение работы сервиса с закрытием всех соединений
- **TLS** - защищённое соединение gRPC
- **gRPC-Gateway** - автоматический маппинг HTTP/JSON запросов в gRPC (описано в `.proto` файле)

## Конфигурация
### Переменные окружения
Создайте `.env` файл на основе примера ниже:
```
# gRPC
GRPC_HOST=0.0.0.0
GRPC_PORT=50051

# Postgres
PG_HOST=pg-user
PG_PORT_INNER=5432
PG_PORT_OUTER=5433
PG_DATABASE_NAME=users_db
PG_USER=user
PG_PASSWORD=password
MIGRATION_DIR=./migrations

# Logger
LOGGER_LEVEL=INFO
LOGGER_AS_JSON=false
LOGGER_ENABLE_OLTP=true

# OpenTelemetry
OTEL_EXPORTER_OTLP_ENDPOINT=otel-collector:4317
OTEL_SERVICE_NAME=user_server
OTEL_ENVIRONMENT=development
OTEL_SERVICE_VERSION=1.0.0
OTEL_METRICS_PUSH_TIMEOUT=1s

# Rate Limiter
RATE_LIMITER_LIMIT=30
RATE_LIMITER_PERIOD=1s

# Kafka
KAFKA_BROKERS=kafka:29092
USER_CREATED_TOPIC_NAME=user.created
USER_DELETED_TOPIC_NAME=user.deleted

# JWT (для валидации токенов из Auth Service)
REFRESH_TOKEN_SECRET=refresh_secret
ACCESS_TOKEN_SECRET=access_secret
```
## API
### gRPC Endpoints
Сервис предоставляет следующие gRPC методы (описаны в `proto/user.proto`):
- `Create` - создание нового пользователя (требует JWT)
- `Get` - получение информации о пользователе 
- `Delete` - удаление пользователя (требует JWT)
- `Update` - обновление пароля (требует JWT)
- `ValidateCredentials` - валидация учетных данных для [Auth Service](https://github.com/WithSoull/AuthService)

## Авторизация
Для операций, требующих авторизации (например, удаление пользователя), используется **JWT токен**. 
- Токены выдаются [Auth Service](https://github.com/WithSoull/AuthService)
- В UserServer используется **TokenVerifier** из [платформенной библиотеки](https://github.com/WithSoull/platform_common) для проверки подписи JWT
- **Политика безопасности:** пользователь может обновить и удалить только свою учетную запись

## Event-Driven
Сервис публикует события в Kafka при изменении состояния пользователей:
### Публикуемые события
| Топик | Событие | Описание |
|-------|---------|----------|
| `user.created` | При создании пользователя | Уведомление других сервисов о новом пользователе |
| `user.deleted` | При удалении пользователя | Каскадное удаление данных в других сервисах |

**Реализация:** используется обёртка Kafka из [платформенной библиотеки](https://github.com/WithSoull/platform_common) с двумя продюсерами.

## Отказоустойчивость
### Rate Limiter
Реализован **Rate Limiter** для защиты от перегрузки:
- Лимит: 30 запросов/сек (конфигурируется через `RATE_LIMITER_LIMIT`)
- Период: 1 секунда (конфигурируется через `RATE_LIMITER_PERIOD`)
- Реализация находится в [платформенной библиотеке](https://github.com/WithSoull/platform_common)
  
## Мониторинг и Observability
### OpenTelemetry (OTEL)
Интегрирован мониторинг из [платформенной библиотеки](https://github.com/WithSoull/platform_common):
- **Traces** - распределённая трассировка запросов
- **Metrics** - метрики производительности
- **Logs** - структурированное логирование

Метрики экспортируются в **OTEL Collector** для дальнейшей обработки.

### Health Check
```
grpcurl -plaintext localhost:50051 grpc.health.v1.Health/Check
```

## Тестирование
Проект является **учебным**, поэтому тестовое покрытие ограничено. Основной фокус - демонстрация подходов к тестированию:
- Используется **minimock** для мокирования зависимостей между слоями
- Написаны примеры unit-тестов для демонстрации изоляции слоёв (handler → service → repository)
- Полное покрытие тестами не реализовано из-за ограничений по времени

## Безопасность
- **TLS** - все gRPC соединения защищены TLS
- **JWT валидация** - проверка подписи токенов из Auth Service
- **Rate Limiting** - защита от DDoS и перегрузки
- **SQL Injection** - использование prepared statements в repository layer
- **Principle of Least Privilege** - пользователь может изменять только свои данные

## Зависимости
### Внешние сервисы
- PostgreSQL 16
- Kafka (для event streaming)
- OTEL Collector (для мониторинга)
- Envoy Proxy (для маршрутизации)

### [Платформенная библиотека](https://github.com/WithSoull/platform_common)
Сервис использует переиспользуемые компоненты из общей платформенной библиотеки:
- TokenVerifier (JWT validation)
- OTEL wrapper
- Rate Limiter
- Kafka wrapper
