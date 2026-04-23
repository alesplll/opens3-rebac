<p align="center">
  <img src="https://img.shields.io/badge/Go-1.24.1-00ADD8?logo=go" alt="Go" />
  <img src="https://img.shields.io/badge/Python-3.12-3776AB?logo=python&logoColor=white" alt="Python" />
  <img src="https://img.shields.io/badge/gRPC-Protobuf-00C7B7?logo=google-cloud&logoColor=white" alt="gRPC" />
  <img src="https://img.shields.io/badge/S3-Compatible%20API-FF9900?logo=amazons3&logoColor=white" alt="S3" />
  <img src="https://img.shields.io/badge/ReBAC-Authorization-6B46C1" alt="ReBAC" />
  <img src="https://img.shields.io/badge/Neo4j-Graph%20DB-008CC1?logo=neo4j&logoColor=white" alt="Neo4j" />
  <img src="https://img.shields.io/badge/PostgreSQL-Metadata-4169E1?logo=postgresql&logoColor=white" alt="PostgreSQL" />
  <img src="https://img.shields.io/badge/Redis-Cache-DC382D?logo=redis&logoColor=white" alt="Redis" />
  <img src="https://img.shields.io/badge/Kafka-Events-231F20?logo=apachekafka&logoColor=white" alt="Kafka" />
  <img src="https://img.shields.io/badge/Docker-Compose-2496ED?logo=docker&logoColor=white" alt="Docker" />
  <img src="https://img.shields.io/badge/Rust-1.87-000000?logo=rust&logoColor=white" alt="Rust" />
  <img src="https://img.shields.io/badge/OpenTelemetry-Tracing-425CC7?logo=opentelemetry&logoColor=white" alt="OpenTelemetry" />
  <img src="https://img.shields.io/badge/Jaeger-Tracing-66CFE3?logo=jaeger&logoColor=white" alt="Jaeger" />
  <img src="https://img.shields.io/badge/Prometheus-Metrics-E6522C?logo=prometheus&logoColor=white" alt="Prometheus" />
  <img src="https://img.shields.io/badge/Grafana-Dashboards-F46800?logo=grafana&logoColor=white" alt="Grafana" />
  <img src="https://img.shields.io/badge/Elasticsearch-Logs-005571?logo=elasticsearch&logoColor=white" alt="Elasticsearch" />
  <img src="https://img.shields.io/badge/Kibana-UI-E8488B?logo=kibana&logoColor=white" alt="Kibana" />
</p>

</p>

<h1 align="center">OpenS3-ReBAC</h1>

<p align="center">
  Распределённое объектное хранилище с S3 API и авторизацией на основе ReBAC
</p>

---

## Что это

**OpenS3-ReBAC** — учебный командный проект: S3-совместимое объектное хранилище (бакеты, объекты, версионирование) с гибкой авторизацией на основе графа отношений (Relationship-Based Access Control).

Клиенты работают через стандартный S3 API (boto3, aws-cli, любой S3 SDK). Внутри — четыре микросервиса, общающихся по gRPC.

## Быстрый старт

Если у вас установлен только Docker и вы хотите просто поднять локально всё, что уже реализовано, используйте пошаговую инструкцию:

- [GETTING_STARTED.md](GETTING_STARTED.md)

Короткая версия:

```bash
cd /путь/до/opens3-rebac
docker compose --profile services up --build -d
```

Важно: команды нужно запускать именно из корня репозитория, где лежит `docker-compose.yml`.

---

## Архитектура

```
Client (HTTP / S3 API)
        │
        ▼
    Gateway :8080          ← единственная точка входа
   /  |  |   \
  /   |  |    \
Auth AuthZ Meta Storage    ← gRPC-сервисы
:50050 :50051 :50052 :50053
  │      │      │
  │    Neo4j  PostgreSQL
  │    Redis
  │
Users :50051               ← управление пользователями
  │
PostgreSQL
         │
       Kafka               ← асинхронные события между сервисами
```

## Сервисы

| Сервис | Стек | Порт | Ответственный |
|---|---|---|---|
| **Gateway** | Go | `:8080` | Макс |
| **AuthZ (ReBAC)** | Python | `:50051` | Алекса |
| **Metadata** | Python | `:50052` | Аня |
| **Data Node** | Go | `:50053` | Илья |
| **Auth** | Go | `:50050` | — |
| **Users** | Go | `:50051` (gRPC) | — |

---

## AuthZ / ReBAC Service

Сервис авторизации отвечает на вопрос: **«может ли `user:alice` выполнить `read` над `object:photos/cat.jpg`?»**

Решение принимается обходом графа отношений в **Neo4j**: пользователь → группы → ресурсы. Результат кэшируется в **Redis** (TTL 30 с). Каждое решение и изменение графа аудитируется через **Kafka**.

```
Gateway → Check(subject, action, object)
              │
              ├─ Redis (cache hit) → ALLOW/DENY
              └─ Neo4j (graph traversal) → ALLOW/DENY → кэш + аудит
```

gRPC API: `Check` · `WriteTuple` · `DeleteTuple` · `Read` · `HealthCheck`

Подробнее: [`services/authz/README.md`](services/authz/README.md)

---

## Auth Service

Сервис аутентификации пользователей. Выдаёт JWT refresh/access токены и валидирует их для других сервисов.

```
Gateway → Auth.Login(email, password)
            │
            ├─ Users (gRPC ValidateCredentials) → OK
            └─ выдаёт refresh token + access token
```

Защита от перебора: Redis хранит счётчик неудачных попыток (`login_attempts:{email}`, TTL 30 с, лимит 6 попыток). Rate limiter: 30 req/s.

gRPC API: `Login` · `GetRefreshToken` · `GetAccessToken` · `ValidateToken` · `HealthCheck`

Подробнее: [`services/auth/README.md`](services/auth/README.md)

---

## Users Service

Сервис управления пользователями. Хранит учётные записи в PostgreSQL, публикует события в Kafka. Является источником истины о credentials — Auth Service обращается к нему при каждом логине.

```
Auth → ValidateCredentials(email, password) → Users
                                                 │
                                              PostgreSQL (bcrypt check)
```

При создании/удалении пользователя публикует события `user.created` / `user.deleted` в Kafka для каскадной обработки в других сервисах.

gRPC API: `Create` · `Get` · `Delete` · `Update` · `ValidateCredentials` · `HealthCheck`

Подробнее: [`services/users/README.md`](services/users/README.md)

---

## Структура репозитория

```
opens3-rebac/
├── proto/                        # Shared gRPC контракты (source of truth)
│   ├── authz/v1/authz.proto      # opens3.authz.v1.PermissionService
│   ├── metadata/v1/metadata.proto # opens3.metadata.v1.MetadataService
│   └── storage/v1/storage.proto  # opens3.storage.v1.DataStorageService
│
├── services/
│   ├── authz/                    # ReBAC authorization engine (Python)
│   ├── metadata/                 # Metadata service (Python)
│   ├── storage/                  # Data Node (Go)
│   ├── gateway/                  # HTTP Gateway (Go)
│   ├── auth/                     # Authentication service (Go)
│   └── users/                    # User management service (Go)
│
├── infra/                        # Docker Compose, K8s манифесты
├── .github/                      # CI/CD workflows
└── docs/                         # Диаграммы, ADR, документация
```

---

## Запуск

Для первого локального запуска используйте пошаговую инструкцию:

- [GETTING_STARTED.md](/Users/grishinid/home/01_Coding/06_pets/opens3-rebac/GETTING_STARTED.md)
- [GETTING_STARTED.md](GETTING_STARTED.md)

Короткие команды через `make`:

```bash
make up-services
make down
make down-volumes
make rebuild
```

> Gateway и Metadata пока не реализованы, поэтому полного S3 flow ещё нет.

### Observability

```bash
make up-observability

docker compose --profile services --profile observability down
```

| UI | Адрес |
|---|---|
| Jaeger | http://localhost:16686 |
| Prometheus | http://localhost:9090 |
| Grafana | http://localhost:3000 |
| Kibana | http://localhost:5601 |
| Neo4j Browser | http://localhost:7474 (neo4j / password123) |

---

## Roadmap

| Фаза | Статус | Что |
|---|---|---|
| **Phase 0** | ✅ Done | Синхронизация, контракты, Docker Compose |
| **Phase 1** | 🔄 In Progress | MVP: PutObject + GetObject end-to-end |
| **Phase 2** | ⏳ | CreateBucket, DeleteBucket, DeleteObject, Kafka, права, версионирование |
| **Phase 3** | ⏳ | Multipart upload, шеринг объектов, S3-совместимость |
| **Phase 4** | ⏳ | Аудит, мониторинг, E2E тесты |
