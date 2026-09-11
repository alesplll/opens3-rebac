# Storage Service: состояние и дальнейший план

Обновлено: 2026-09-11.

Этот документ заменяет исторический план от 2026-04-07. Текущий контракт подробно
описан в [`services/storage/README.md`](../services/storage/README.md), устройство
filesystem — в [`storage-fs-architecture.md`](storage-fs-architecture.md).

## Реализовано

- gRPC server на Go с client-streaming StoreObject/UploadPart и
  server-streaming RetrieveObject;
- immutable blob IDs, sharded final paths и temporary files;
- full/range retrieval через offset/length;
- идемпотентный DeleteObject;
- multipart initiate, overwrite part, complete, abort и completion metadata;
- empty blob и large/component tests;
- custom filesystem health и standard gRPC health;
- OpenTelemetry instrumentation.

Blob ID создаётся service до repository write. Пустой blob допустим. Final file
становится видимым после rename. Multipart final blob ID в текущей реализации
совпадает с upload ID; cleanup session выполняется внутри Storage после commit.

## Ограничения текущего commit

`fsync` файла и rename на одной filesystem дают атомарную видимость final path.
Код не делает fsync каталога, поэтому не заявляется полная power-loss durability.
Раздельные `DATA_DIR` и staging filesystem нарушат предпосылку atomic rename.

Completion metadata нужна для retry Complete после потерянного ответа. Её нельзя
удалять сразу после Metadata finalize, пока Metadata или другой durable registry
не умеет вернуть сохранённый terminal result.

## Следующие работы

### 1. Gateway integration

Gateway должен передавать HTTP body в StoreObject с bounded buffering, после
durable Storage response зарегистрировать committed Metadata version и только
затем вернуть клиенту успех. Следующий GET/HEAD/LIST после успеха должен видеть
эту version.

Для нового key authorization policy проверяется на bucket; overwrite policy нужно
явно согласовать для bucket/object. Нельзя использовать `object:{bucket}` как
подмену bucket entity.

### 2. Multipart orchestration

Gateway/Metadata связывают upload ID с bucket/key/initiator и Storage session.
Part number и max size проверяются при upload; минимум 5 MiB проверяется на
Complete для всех выбранных parts кроме последней. External multipart ETag
рассчитывается согласно выбранному checksum mode отдельно от Storage full-blob
MD5.

### 3. Metadata lifecycle

Если сохраняется простой sync flow:

```text
Storage durable commit → Metadata committed version → HTTP success
```

Если вводится pending/finalize:

```text
durable intent + operation/version ID
→ Storage durable commit
→ Metadata atomic finalize/current pointer
→ HTTP success
```

Асинхронный Kafka event может участвовать внутри, но клиентский success barrier не
может предшествовать видимости committed version. Retry использует operation ID,
не ETag.

### 4. Cleanup events

Требуется единый контракт topic → producer → consumer → schema. Текущий Storage
ещё не содержит Kafka consumer. Рекомендуемые logical names должны быть утверждены
один раз; старые варианты `object-blob-stored`/`object-delete-requested` не
смешиваются с `object-stored`/`object-deleted`.

Outbound DB events Metadata должны проходить transactional outbox. Consumers
работают at-least-once и адресуют конкретные resource generation/version/blob IDs.
Delete marker не создаёт команду удаления historical blobs; permanent version
delete и GC — отдельный flow.

### 5. Распределённое хранение

До нескольких nodes нужно изменить идентификацию реплик: текущий StoreObject не
принимает coordinator-selected blob ID. Quorum, fencing, placement state и repair
описаны в [`distributed-storage-architecture.md`](distributed-storage-architecture.md).

## Проверки готовности

- `go test ./...` из `services/storage`;
- component upload/retrieve/delete и multipart retry;
- одинаковый terminal result после потерянного ответа;
- немедленная Metadata visibility после внешнего success;
- orphan reconciliation и version-aware cleanup;
- filesystem-specific crash tests до заявления power-loss durability;
- multi-node failure/partition tests до заявления replication guarantees.