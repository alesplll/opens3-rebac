# Storage Service Phase 2 — Fix Plan

Дата: 2026-04-07

---

## Контекст

Во время ревью ветки `feat/storage/phase2_multipart_service` по реализации Phase 2 из `docs/storage-service-implementation-plan.md` были найдены несколько проблем в multipart flow и связанных обработчиках.

Ниже — план исправлений, отсортированный по приоритету.

---

## P0 — исправить до merge

### 1. Вернуть настоящий streaming в `UploadPart`

### Проблема

Сейчас `UploadPart` в handler сначала целиком буферизует весь входящий part в `bytes.Buffer`, и только потом передаёт его в service/repository.

Это ломает ключевое свойство streaming upload:
- потребление памяти растёт вместе с размером part
- исчезает backpressure на запись в filesystem
- большие multipart upload могут приводить к OOM или сильным memory spikes

### Где

- `services/storage/internal/handler/storage/upload_part.go`

### Что сделать

- заменить `bytes.Buffer` на streaming reader по аналогии с `StoreObject`
- переиспользовать существующий `chunkReader` или выделить общий helper для client-streaming RPC
- передавать reader в `service.UploadPart(...)` без промежуточного накопления всего part в памяти

### Что проверить тестами

- unit test handler-а, который проверяет, что service получает объединённый stream из нескольких чанков
- test на пустой stream
- component test с multipart upload из нескольких чанков
- по возможности test, который подтверждает отсутствие необходимости читать весь part в память перед записью

---

### 2. Убрать риск orphan blob при `CompleteMultipartUpload`

### Проблема

В `AssembleParts` итоговый blob уже считается записанным после `os.Rename(temp, final)`, но затем вызывается `CleanupMultipart`. Если cleanup вернёт ошибку, наружу пойдёт failure, хотя blob уже лежит на диске.

Дополнительно service генерирует новый `blob_id` на каждый `CompleteMultipartUpload`, поэтому retry после такого сбоя может:
- создать второй blob
- оставить первый blob без ссылки из Metadata
- сделать flow неидемпотентным и засорить storage orphan-файлами

### Где

- `services/storage/internal/repository/storage/multipart.go`
- `services/storage/internal/service/storage/complete_multipart.go`

### Что сделать

- разделить семантику:
  - успешная сборка blob
  - best-effort cleanup multipart session
- не возвращать клиенту ошибку после успешного `Rename`, если проблема только в cleanup временных multipart файлов
- если cleanup критичен для контракта, тогда нужен compensating action:
  - удалить уже собранный blob при ошибке cleanup
  - или сделать complete идемпотентным на уровне `upload_id -> blob_id`
- отдельно решить и зафиксировать ожидаемую retry-semantics для `CompleteMultipartUpload`

### Предпочтительный вариант

Для текущей архитектуры проще и безопаснее:
- считать `CompleteMultipartUpload` успешным после атомарного commit итогового blob
- cleanup временной директории делать best-effort с логированием
- позже добавить background cleanup/TTL для зависших multipart session

### Что проверить тестами

- unit/component test на сценарий: assemble success + cleanup failure
- test на повторный `CompleteMultipartUpload` после частичного сбоя
- test, подтверждающий отсутствие duplicate blob при retry

---

## P1 — важно исправить в этой же серии или сразу после

### 3. Валидировать консистентность `upload_id` и `part_number` во всех сообщениях stream-а

### Проблема

В `UploadPart` значения `upload_id` и `part_number` берутся только из первого сообщения. Все последующие сообщения молча игнорируются, даже если клиент прислал другие значения.

Это создаёт риск тихой порчи данных при buggy client/gateway:
- чанки от другого `upload_id` могут попасть в текущий part
- чанки с другим `part_number` будут записаны как будто это тот же part

### Где

- `services/storage/internal/handler/storage/upload_part.go`

### Что сделать

- на первом сообщении зафиксировать `upload_id` и `part_number`
- на каждом следующем сообщении проверять, что они не меняются
- при несовпадении возвращать `InvalidArgument`

### Что проверить тестами

- unit test handler-а на mismatch `upload_id`
- unit test handler-а на mismatch `part_number`

---

### 4. Сделать temp file strategy устойчивой к retry после crash/interruption

### Проблема

Сейчас временный файл всегда создаётся как `finalPath + ".tmp"` с `O_EXCL`. Если процесс упал между созданием temp-файла и cleanup, следующий retry той же операции может начать стабильно падать, потому что старый `.tmp` уже существует.

Особенно неприятно это для multipart part upload:
- retry того же `upload_id/part_number` может перестать работать
- recovery потребует `AbortMultipartUpload` или ручной cleanup на диске

### Где

- `services/storage/internal/repository/storage/write_helpers.go`
- косвенно `services/storage/internal/repository/storage/multipart.go`

### Что сделать

- перейти на уникальные temp filenames, например через `os.CreateTemp(dir, base+".*")`
- commit оставлять через `os.Rename(temp, final)`
- cleanup делать по конкретному temp path

### Что проверить тестами

- test на retry после оставшегося `.tmp`
- test на overwrite/retry одного и того же multipart part

---

## P2 — рефакторинг и hardening

### 5. Унифицировать client-streaming handlers

### Проблема

`StoreObject` уже использует потоковое чтение через `chunkReader`, а `UploadPart` реализован отдельно и сейчас делает это хуже. Логика чтения первого сообщения и сборки stream-а дублируется концептуально.

### Что сделать

- вынести общий helper для client-streaming request body
- унифицировать поведение:
  - first message required
  - empty stream -> `io.ErrUnexpectedEOF`
  - метаданные первого сообщения фиксируются и валидируются

### Ожидаемый эффект

- меньше расхождений между `StoreObject` и `UploadPart`
- меньше шансов повторно сломать streaming semantics

---

### 6. Добавить явные тесты на retry semantics multipart lifecycle

### Проблема

Текущие тесты хорошо покрывают happy path и checksum mismatch, но почти не покрывают operational edge cases:
- retry upload part
- retry complete
- отмена после частично записанных данных
- поведение после context cancellation

### Что сделать

Добавить unit/component tests на:
- повторную загрузку одного и того же part
- complete после missing part
- complete после cleanup failure
- abort после partially uploaded part
- context cancellation во время записи part и assembly

---

## Рекомендуемый порядок работ

1. Починить `UploadPart` streaming и валидацию metadata в stream.
2. Исправить semantics `CompleteMultipartUpload` вокруг post-rename cleanup.
3. Перевести temp files на уникальные имена.
4. Добрать unit/component tests на retry и failure paths.
5. После этого сделать маленький refactor client-streaming helpers.

---

## Definition of Done

Исправления можно считать завершёнными, когда:

- `UploadPart` больше не буферизует весь part в памяти
- `CompleteMultipartUpload` не оставляет orphan blob при cleanup/retry сценариях
- retry multipart операций устойчив к оставшимся `.tmp` файлам
- mismatch metadata внутри одного upload stream не проходит молча
- добавлены тесты на failure/retry paths, а не только на happy path
