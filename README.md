<p align="center">
  <img src="https://img.shields.io/badge/Go-1.24.1-00ADD8?logo=go" alt="Go 1.24.1" />
  <img src="https://img.shields.io/badge/Python-3.12-3776AB?logo=python&logoColor=white" alt="Python 3.12" />
  <img src="https://img.shields.io/badge/Rust-1.95-000000?logo=rust&logoColor=white" alt="Rust 1.95" />
  <img src="https://img.shields.io/badge/gRPC-Protobuf-00C7B7?logo=google-cloud&logoColor=white" alt="gRPC and Protobuf" />
</p>

<h1 align="center">OpenS3-ReBAC</h1>

<p align="center">
  Учебный монорепозиторий сервисов для объектного хранилища с ReBAC-авторизацией.
</p>

## Текущее состояние

В репозитории реализованы внутренние gRPC-сервисы Auth, Users, AuthZ, Metadata,
Storage и Quota. Внешний HTTP Gateway ещё не реализован, поэтому текущий checkout
не предоставляет готовый S3 endpoint для `aws-cli`, boto3 и других стандартных S3
клиентов. Диаграммы распределённого хранения, Gateway, репликации и части Kafka-flow
в `docs/` и Wiki описывают проектируемую архитектуру.

| Сервис | Реализация | Порт на хосте | Основные зависимости |
|---|---|---:|---|
| Auth | Go | 50050 | Users, Redis |
| AuthZ | Python | 50051 | Neo4j, Redis, Kafka |
| Metadata | Go | 50052 | PostgreSQL, Kafka |
| Storage | Go | 50053 | локальная файловая система |
| Users | Go | 50054 | PostgreSQL |
| Quota | Rust | 50055 | Redis |
| Gateway | планируется | 8080 | внутренние gRPC-сервисы |

Сервис Storage хранит immutable blobs и поддерживает внутренние RPC для обычной и
multipart-загрузки. Metadata хранит buckets, objects и versions. Наличие этих RPC
само по себе не означает полной S3-совместимости: внешний HTTP-контракт, SigV4 и
end-to-end orchestration входят в будущий Gateway.

## Быстрый старт

Нужны Docker и Docker Compose plugin. Из корня репозитория:

```bash
docker compose --profile services up --build -d
docker compose ps
```

Подробная инструкция: [GETTING_STARTED.md](GETTING_STARTED.md).

Через Makefile доступны основные команды:

```bash
make up-services
make up-e2e
make test-metadata-integration
make test-metadata-integration-local
make up-observability
make down
make down-volumes  # удаляет локальные данные volumes
make generate
```

## Реализованные границы сервисов

### Auth

`Login` проверяет credentials через Users и возвращает refresh token.
`GetAccessToken` отдельно выпускает access token. Новый refresh token сейчас не
отзывает старый: server-side revocation ещё не реализован. Redis хранит счётчики
неудачных входов, а не реестр JWT-сессий. Rate limiter действует внутри одного
процесса.

### Users

Users хранит учётные записи в PostgreSQL и предоставляет Create, Get, Update,
UpdatePassword, Delete и ValidateCredentials. Сервис рассчитан на внутреннюю сеть:
текущий gRPC server не проверяет JWT и не ограничивает переданный `user_id`
идентичностью вызывающего. Kafka-события пользователей сейчас не публикуются.

### AuthZ

AuthZ проверяет прямые `HAS_PERMISSION` и разрешения групп, достижимых через
`MEMBER_OF`; более высокий permission level включает нижние. Ребро `PARENT_OF`
можно хранить, но текущая проверка не наследует по нему разрешения bucket → object.
Кэш решений имеет TTL. Kafka invalidator не входит в основной Compose-процесс, а
его текущие hints не дают строгой гарантии немедленного отзыва всех зависимых
решений.

### Metadata

Metadata предоставляет каталог buckets/objects/versions и публикует события
удаления после изменения БД. До внедрения transactional outbox это best-effort
граница: успешное изменение БД и публикация в Kafka не атомарны.

### Storage

Storage записывает blob во временный файл, синхронизирует файл и атомарно
переименовывает его на той же файловой системе. Это обеспечивает атомарную
видимость имени, но текущий код не синхронизирует каталог и не обещает полную
устойчивость к потере питания. Репликация и placement service пока не реализованы.

### Quota

`CheckQuota` одновременно проверяет и резервирует положительную дельту. После
успешной операции ту же дельту нельзя повторно передавать в `UpdateUsage`; при
неуспехе резерв надо компенсировать отрицательной дельтой. Текущая атомарность
ограничена одной записью и одним процессом Quota, поэтому сервис пока рассчитан на
одну активную реплику.

## Структура репозитория

```text
services/               # auth, users, authz, metadata, storage, quota
shared/api/              # исходные protobuf-контракты
shared/pkg/go/           # сгенерированный Go-код
shared/pkg/py/           # сгенерированный Python-код
shared/pkg/go-kit/       # общие Go-компоненты
shared/pkg/py_kit/       # общие Python-компоненты
shared/pkg/rust-kit/     # общие Rust-компоненты
infra/                   # observability и инфраструктурные настройки
e2e/                     # общая инфраструктура интеграционных тестов
docs/                    # актуальные и датированные проектные документы
```

Сгенерированные файлы в `shared/pkg/go` и `shared/pkg/py` не редактируются
вручную. После изменения `shared/api` используйте `make generate`.

## Observability

```bash
make up-observability
```

| UI | Адрес |
|---|---|
| Jaeger | <http://localhost:16686> |
| Prometheus | <http://localhost:9090> |
| Grafana | <http://localhost:3000> |
| Kibana | <http://localhost:5601> |
| Neo4j Browser | <http://localhost:7474> |

Стандартный gRPC Health сообщает состояние процесса, установленное приложением.
Он не во всех сервисах проверяет доступность базы или диска. У Storage есть
отдельный custom HealthCheck для файлового каталога.

## Статус roadmap

Phase 0 (контракты, Compose и базовые сервисы) завершена частично в текущем
checkout. Ближайшая цель — реализовать Gateway и согласовать end-to-end запись с
сильной read-after-write видимостью. Versioning, multipart orchestration,
репликация, надёжный outbox и полная S3-совместимость остаются дальнейшими этапами.

## Документация

- [Карта всей документации](docs/README.md)
- [Первый запуск](GETTING_STARTED.md)
- [Auth](services/auth/README.md)
- [Users](services/users/README.md)
- [AuthZ](services/authz/README.md)
- [Metadata](services/metadata/README.md)
- [Storage](services/storage/README.md)
- [Quota](services/quota/README.md)
- [GitHub Wiki](https://github.com/alesplll/opens3-rebac/wiki)

## Разработка и вклад

[AGENTS.md](AGENTS.md) фиксирует структуру слоёв, стиль тестов и правила изменения
документации. Service README содержит полный локальный сценарий, env и gRPC examples.
Go modules запускаются отдельно; repository tests Metadata используют отдельную
PostgreSQL из `e2e/`. Для генерации нужен `protoc`, затем `make install-deps` и
`make generate`; Rust генерирует свои bindings при Cargo build.

Цель следующих этапов — связать сервисы в объектное хранилище с управляемым доступом,
версиями и восстановлением после сбоев. [Планы](docs/README.md#проектируемая-архитектура)
содержат варианты реализации и критерии готовности, а не показатели уже достигнутой
S3-совместимости, производительности или production availability.
