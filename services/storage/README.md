# Storage Service

Go gRPC-сервис immutable blob storage. Он работает с `blob_id` и не знает о S3
buckets, object keys, users или правах доступа.

## API

Source of truth: `shared/api/storage/v1/storage.proto`.

| RPC | Тип | Назначение |
|---|---|---|
| `StoreObject` | client stream | сохранить blob, вернуть UUID и full-content MD5 |
| `RetrieveObject` | server stream | читать blob целиком или по offset/length |
| `DeleteObject` | unary | идемпотентно удалить blob |
| `InitiateMultipartUpload` | unary | создать multipart session/upload ID |
| `UploadPart` | client stream | сохранить/перезаписать part |
| `CompleteMultipartUpload` | unary | проверить список parts и собрать blob |
| `AbortMultipartUpload` | unary | идемпотентно очистить session |
| `HealthCheck` | unary | проверить filesystem repository |

Client-streaming StoreObject/UploadPart используют первое сообщение `header`,
затем сообщения `chunk`. Пустой объект поддерживается.

## Файловый commit

Обычная запись получает blob ID в service до начала repository write. Repository
пишет staging file, вызывает `Sync`, закрывает его и выполняет rename в final path.
Final file появляется после rename, а не при первом chunk.

Atomic rename гарантируется только внутри одной filesystem. Текущий код не делает
fsync каталога после rename, поэтому документация не обещает полную durability при
аварийной потере питания.

Multipart upload создаёт session заранее. В текущем repository итоговый blob ID
совпадает с upload ID. Complete сохраняет completion metadata для retry и затем
best-effort очищает session внутри Storage; он не ждёт callback от Metadata.

## S3 и multipart

Storage возвращает MD5 полного собранного blob. Это внутренний checksum, а не
универсальный внешний multipart ETag. Gateway должен отдельно реализовать выбранную
S3 checksum/ETag semantics.

S3 minimum part size проверяется на Complete для всех выбранных частей, кроме
последней: при отдельном UploadPart ещё неизвестно, какая часть окажется последней.
Storage валидирует свой внутренний контракт; полная S3-валидация относится к
Gateway/Metadata orchestration.

## Что пока не реализовано

- репликация blob на несколько Storage nodes;
- placement service и consistent hashing;
- Kafka consumer для object cleanup;
- внешний S3 HTTP API;
- durable связь Storage completion с Metadata finalize.

Нельзя вызвать StoreObject на нескольких узлах и ожидать один blob ID: текущий
request не принимает заранее выбранный ID, а каждый service сам создаёт UUID.

## Конфигурация

Основные значения находятся в `services/storage/.env`:

```dotenv
GRPC_PORT=50053
DATA_DIR=/data/blobs
MULTIPART_DIR=/data/multipart
```

Для гарантированного rename staging/final paths должны находиться на одной
filesystem.

## Запуск

Из корня репозитория:

```bash
docker compose --profile services up --build -d storage
docker compose logs -f storage
```

Локально:

```bash
cd services/storage
go run ./cmd/server
```

## Proto generation

Из корня:

```bash
make generate-storage-go
make generate-storage-py
```

Либо сгенерировать все bindings:

```bash
make generate
```

`go.work` подменяет опубликованные module versions локальными каталогами во время
работы в workspace. При этом `services/storage/go.mod` всё равно содержит обычную
module dependency на shared package.

## Тесты

```bash
cd services/storage
go test ./...
```

Инвентаризация тестов: [tests.md](tests.md). Standard gRPC Health выставляется в
SERVING при старте; custom `DataStorageService.HealthCheck` отдельно проверяет
filesystem. Эти два endpoint нельзя считать одной и той же readiness-проверкой.