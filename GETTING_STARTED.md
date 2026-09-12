# Getting Started

Эта инструкция поднимает текущие внутренние сервисы проекта. Внешний S3 Gateway
ещё не реализован, поэтому после запуска не появится S3 HTTP endpoint.

## Требования

- Docker с Compose plugin;
- запуск команд из корня репозитория, где находится `docker-compose.yml`.

Проверка:

```bash
docker --version
docker compose version
```

## Запуск

```bash
docker compose --profile services up --build -d
docker compose ps
```

Профиль `services` запускает:

- одноразовые `migrator-users` и `migrator-metadata`;
- `users`, `auth`, `authz`, `metadata`, `storage`, `quota`;
- PostgreSQL для Users и Metadata, Redis, Neo4j, ZooKeeper и Kafka.

Миграторы в норме завершаются с кодом 0. Долгоживущие контейнеры должны быть в
состоянии Up/healthy согласно их Compose healthcheck.

## Порты на хосте

| Компонент | Адрес |
|---|---|
| Auth | `localhost:50050` |
| AuthZ | `localhost:50051` |
| Metadata | `localhost:50052` |
| Storage | `localhost:50053` |
| Users | `localhost:50054` |
| Quota | `localhost:50055` |
| PostgreSQL Users | `localhost:5432` |
| PostgreSQL Metadata | `localhost:5433` |
| Redis | `localhost:6379` |
| Kafka external listener | `localhost:9092` |
| Neo4j HTTP | `localhost:7474` |
| Neo4j Bolt | `localhost:7687` |

Контейнеры обращаются к Kafka через `kafka:29092`, а не через внешний listener
9092.

## Проверка и диагностика

```bash
docker compose ps
docker compose logs -f metadata
docker compose logs -f quota
docker compose logs -f storage
```

Для другого сервиса замените имя в последней команде. Типичные причины сбоя:
занятый порт, не запущенный Docker daemon, недостаток ресурсов или неготовая
зависимость.

Стандартный health endpoint проверяет заявленный приложением статус процесса и не
обязательно выполняет запрос к базе/диску. Для Storage filesystem проверяет
отдельный RPC `DataStorageService.HealthCheck`.

## Остановка

```bash
docker compose --profile services down
```

Удаление volumes также удаляет локальные данные:

```bash
docker compose --profile services down -v
```

## Makefile

Если установлен `make`:

```bash
make up-services
make down
make down-volumes
make rebuild
```

Для отдельной PostgreSQL интеграционных тестов Metadata:

```bash
make up-e2e
make test-metadata-integration
make down-e2e
```

Или одним запуском:

```bash
make test-metadata-integration-local
```

Observability поднимается отдельным профилем:

```bash
make up-observability
```

## Разработка вне контейнера

| Компонент | Нужные инструменты | Инструкция |
|---|---|---|
| Go services | Go 1.24.1; зависимости в Docker | [Auth](services/auth/README.md#запуск), [Users](services/users/README.md#запуск), [Metadata](services/metadata/README.md#запуск), [Storage](services/storage/README.md#запуск) |
| AuthZ | Python 3.12, venv, pip | [Setup и отдельный invalidator](services/authz/README.md#запуск) |
| Quota | Rust (Docker build использует 1.95), protoc, Redis | [Запуск и tests](services/quota/README.md#запуск) |
| Ручные RPC | grpcurl | раздел «Примеры использования» каждого сервиса |
| Генерация proto | protoc, Go, Python/pip, make | команды ниже |

Go `.env` читается относительно папки запуска, Python Config — только из окружения,
Rust использует dotenvy от текущей папки. Для local process адреса Docker DNS
заменяются на localhost и опубликованные порты. Если контейнер сервиса уже запущен,
остановите именно его, чтобы освободить порт; зависимости оставьте работающими.

## Изменение protobuf

Из корня:

```bash
make install-deps
make generate
```

`install-deps` кладёт Go plugins и Python grpcio-tools в `bin/`; сам `protoc`
нужно установить отдельно. Go/Python clients генерируются в `shared/pkg/go` и
`shared/pkg/py`. Для отдельного сервиса есть targets `generate-<service>-go/py`;
у Users имя target — `generate-user-go/py`. Rust Quota генерирует bindings через
`build.rs` при сборке и также требует `protoc`.

## Что проверять при первом запуске

1. `docker compose ps -a`: миграторы должны завершиться с кодом 0; healthcheck
   есть не у каждого приложения, отсутствие метки healthy не равно ошибке.
2. Standard gRPC Health показывает статус процесса. Custom HealthCheck у Metadata
   проверяет PostgreSQL/Kafka, AuthZ — Neo4j/Redis, Storage — filesystem, Quota — Redis.
3. `make up-observability` поднимает Collector и UI. Ошибка экспорта telemetry
   при отсутствии Collector не является доказательством ошибки бизнес-RPC.
4. При ошибке credentials БД проверьте соответствие `.env` и существующего volume:
   новые значения env не переинициализируют уже созданную БД. `down -v` удаляет данные.
5. Проверьте реальный запрос из README сервиса. Успех health сам по себе не является
   end-to-end проверкой записи объекта.

[Карта документации](docs/README.md) связывает сервисы, архитектурные планы и аудиты.
