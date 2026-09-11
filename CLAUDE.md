# CLAUDE.md — OpenS3-ReBAC

Этот файл даёт AI-ассистентам короткий, проверяемый контекст репозитория. Более
подробные правила разработки находятся в [AGENTS.md](AGENTS.md).

## Что есть в текущем checkout

| Сервис | Язык | Порт | Состояние |
|---|---|---:|---|
| Auth | Go 1.24.1 | 50050 | реализован |
| AuthZ | Python 3.12 | 50051 | реализован |
| Metadata | Go 1.24.1 | 50052 | реализован |
| Storage | Go 1.24.1 | 50053 | реализован |
| Users | Go 1.24.1 | 50054 | реализован |
| Quota | Rust 1.95 | 50055 | реализован |
| Gateway | Go, план | 8080 | отсутствует |

Из-за отсутствия Gateway проект пока не предоставляет внешний S3 HTTP API.
Репликация Storage, placement service, полный versioning lifecycle и часть
event-driven cleanup находятся на уровне проекта/backlog.

## Код и контракты

- protobuf source of truth: `shared/api/<service>/v1/*.proto`;
- generated Go: `shared/pkg/go`;
- generated Python: `shared/pkg/py`;
- общие библиотеки: `shared/pkg/go-kit`, `shared/pkg/py-kit`,
  `shared/pkg/rust-kit`;
- Go-сервисы: `services/{auth,users,metadata,storage}`;
- Python AuthZ: `services/authz`;
- Rust Quota: `services/quota`.

Generated stubs не редактируются вручную. После изменения proto запускается
`make generate`.

## Важные границы текущей семантики

### Auth

`Login` возвращает refresh token. Access token выдаёт отдельный
`GetAccessToken`. `GetRefreshToken` выпускает новый token, но пока не отзывает
старый. JWT содержит строковый UUID в claim `user_id` и `token_type`; `sub` не
является текущим источником ID. Redis используется для login-attempt counters.

### Users

Users — внутренний gRPC-сервис. Текущий server не проверяет JWT/TLS identity и не
ограничивает переданный `user_id` вызывающим пользователем. События
`user.created`/`user.deleted` сейчас не публикуются.

### AuthZ

Публичные relations: `MEMBER_OF`, `HAS_PERMISSION`, `PARENT_OF`. Для владельца
используйте `HAS_PERMISSION` с уровнем `ADMIN`; `OWNER_OF` — только legacy-чтение
в store и не принимается публичным enum.

Permission levels: `admin > delete > create > write > read`; более высокий
уровень включает нижние. Текущий graph check следует через `MEMBER_OF` к прямому
`HAS_PERMISSION` на запрошенном ресурсе. Он не наследует bucket permission на
object через `PARENT_OF`.

Формат ресурсов:

- `user:<uuid>` и `group:<name>`;
- `bucket:<name>`;
- `object:<bucket>/<key>`.

Для нового объекта нельзя проверять вымышленный `object:<bucket>`. До реализации
Gateway необходимо отдельно согласовать policy: создание нового key обычно
проверяется на bucket, overwrite — на существующем object и/или bucket согласно
выбранной модели.

AuthZ cache имеет TTL 30 секунд. Основной Compose запускает gRPC process без
отдельного invalidation consumer; имеющиеся hints не обеспечивают строгий
немедленный отзыв всех group-derived решений.

### Metadata и внешний успех записи

Metadata хранит buckets, objects и versions в PostgreSQL. Текущая публикация
delete events выполняется после DB change и до внедрения outbox является
best-effort.

Независимо от внутреннего sync/async протокола будущий успешный PutObject/Complete
должен возвращаться только после того, как следующий GET/HEAD/LIST сможет увидеть
committed version. `Eventually` допустим для physical GC и readiness зависимостей,
но не как замена read-after-write контракту объекта.

ETag не используется как operation ID. Независимые PUT одинакового содержимого
могут создавать разные versions. Для безопасного retry внутренних мутаций нужен
отдельный стабильный operation ID и сохранённый результат.

### Storage

Storage хранит blobs по ID на локальной файловой системе. При записи он использует
temporary file, `Sync` файла и rename. Это даёт атомарную видимость rename только
на одной filesystem; каталог пока не синхронизируется, поэтому нельзя обещать
полную power-loss durability. Репликация/placement не реализованы.

Multipart session создаётся и очищается внутри Storage. Минимальный S3 part size
можно проверить только при Complete для всех выбранных частей, кроме последней.
Внешний multipart ETag и внутренний full-blob MD5 — разные понятия.

### Quota

`CheckQuota` выполняет check-and-reserve. После успешной операции не вызывайте
`UpdateUsage` с той же положительной дельтой; при неуспехе делайте компенсацию.
Без operation/reservation ID retries не являются exactly-once. Текущий in-memory
учёт согласован только внутри одного процесса, поэтому несколько активных Quota
replicas не дают строгий общий лимит.

## Команды

```bash
make up-services
make up-observability
make generate
make test-metadata-integration-local
make down
```

Для unit tests запускайте `go test ./...` из нужного Go module,
`python3 -m pytest tests/unit -v` из `services/authz` и
`cargo test -p quota-service` из корня.

## Документы

Документы с датой и branch name могут быть историческими планами. Не используйте
их секции «текущее состояние» без сверки с кодом. В актуальном тексте всегда
разделяйте:

1. уже реализованный runtime;
2. утверждённый внешний контракт;
3. проектируемую архитектуру и acceptance criteria.