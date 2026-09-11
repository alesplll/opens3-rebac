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