<p align="center">
  <img src="https://img.shields.io/badge/Go-1.23-00ADD8?logo=go&logoColor=white" alt="Go" />
  <img src="https://img.shields.io/badge/Python-3.12-3776AB?logo=python&logoColor=white" alt="Python" />
  <img src="https://img.shields.io/badge/gRPC-Protobuf-00C7B7?logo=google-cloud&logoColor=white" alt="gRPC" />
  <img src="https://img.shields.io/badge/S3-Compatible%20API-FF9900?logo=amazons3&logoColor=white" alt="S3" />
  <img src="https://img.shields.io/badge/ReBAC-Authorization-6B46C1" alt="ReBAC" />
  <img src="https://img.shields.io/badge/Neo4j-Graph%20DB-008CC1?logo=neo4j&logoColor=white" alt="Neo4j" />
  <img src="https://img.shields.io/badge/PostgreSQL-Metadata-4169E1?logo=postgresql&logoColor=white" alt="PostgreSQL" />
  <img src="https://img.shields.io/badge/Redis-Cache-DC382D?logo=redis&logoColor=white" alt="Redis" />
  <img src="https://img.shields.io/badge/Kafka-Events-231F20?logo=apachekafka&logoColor=white" alt="Kafka" />
  <img src="https://img.shields.io/badge/Docker-Compose-2496ED?logo=docker&logoColor=white" alt="Docker" />
</p>

<h1 align="center">OpenS3-ReBAC</h1>

<p align="center">
  Распределённое объектное хранилище с S3 API и авторизацией на основе ReBAC
</p>

---

## Что это

**OpenS3-ReBAC** — учебный командный проект: S3-совместимое объектное хранилище (бакеты, объекты, версионирование) с гибкой авторизацией на основе графа отношений (Relationship-Based Access Control).

Клиенты работают через стандартный S3 API (boto3, aws-cli, любой S3 SDK). Внутри — четыре микросервиса, общающихся по gRPC.

---

## Архитектура

```
Client (HTTP / S3 API)
        │
        ▼
    Gateway :8080          ← единственная точка входа
   /    |    \
  /     |     \
AuthZ  Meta  Storage       ← gRPC-сервисы
:50051 :50052 :50053
  │      │
Neo4j  PostgreSQL
Redis
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
│   └── gateway/                  # HTTP Gateway (Go)
│
├── infra/                        # Docker Compose, K8s манифесты
├── .github/                      # CI/CD workflows
└── docs/                         # Диаграммы, ADR, документация
```

---

## Roadmap

| Фаза | Статус | Что |
|---|---|---|
| **Phase 0** | ✅ Done | Синхронизация, контракты, Docker Compose |
| **Phase 1** | 🔄 In Progress | MVP: PutObject + GetObject end-to-end |
| **Phase 2** | ⏳ | CreateBucket, DeleteBucket, DeleteObject, Kafka, права, версионирование |
| **Phase 3** | ⏳ | Multipart upload, шеринг объектов, S3-совместимость |
| **Phase 4** | ⏳ | Аудит, мониторинг, E2E тесты |
