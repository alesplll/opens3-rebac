# Storage Service FS Architecture

Дата: 2026-04-08

---

## Цель

Storage Service должен оставаться **blob-хранилищем**, а не хранилищем S3-объектов.

Это означает:
- Storage **не знает** `bucket` и `key`
- Storage **не хранит** объектные метаданные как источник истины
- Storage работает только с:
  - `blob_id`
  - `upload_id`
  - part-файлами multipart upload
  - локальным staging / cleanup lifecycle

Источник истины о существовании объекта находится в Metadata Service.

---

## Основные принципы

1. Данные, которые ещё не финализированы, не должны лежать вперемешку с финальными blob.
2. Финальный blob должен быть immutable и адресоваться только по `blob_id`.
3. Multipart и single-part upload должны проходить через staging-зону.
4. Очистка orphan/stale данных должна быть штатной частью системы.
5. Storage не должен зависеть от структуры object key вроде `photos/2026/cat.jpg`.

---

## Рекомендуемая структура директорий

```text
/data/
  blobs/
    ab/
      abcd1111-2222-3333-4444-555566667777
    c4/
      c4ef1111-2222-3333-4444-555566667777

  staging/
    uploads/
      <upload_id>/
        object.bin
        manifest.json
        part_00001
        part_00002
        part_00003

  gc/
    trash/
      2026-04-08/
        550e8400-e29b-41d4-a716-446655440000
```

---

## Назначение зон

### `blobs/`

Хранит только **финальные immutable blob**, уже опубликованные или готовые к публикации.

Формат пути:

```text
/data/blobs/{blob_id[0:2]}/{blob_id}
```

Зачем шардирование:
- не складывать сотни тысяч файлов в один каталог
- упростить дальнейшее масштабирование и background scan

### `staging/uploads/<upload_id>/`

Хранит временные данные, которые ещё не стали опубликованным blob.

Для single-part upload:
- `object.bin`
- `manifest.json`

Для multipart upload:
- `part_00001`
- `part_00002`
- ...
- `manifest.json`

`manifest.json` полезен для локальной диагностики и GC. Минимально в нём можно хранить:
- `upload_id`
- `created_at`
- `content_type`
- `expected_parts`
- `state`

### `gc/trash/`

Опциональная промежуточная зона перед физическим удалением.

Нужна если вы хотите:
- отложенное удаление
- мягкую защиту от ошибочного purge
- более удобный reconcile/debug flow

Если такой режим не нужен, можно удалять blob сразу.

---

## Жизненный цикл данных

### Single-part PutObject

1. Gateway/Metadata создаёт upload intent.
2. Storage пишет тело в `staging/uploads/<upload_id>/object.bin`.
3. После успешной записи и валидации создаётся финальный `blob_id`.
4. Blob атомарно переносится в `blobs/{shard}/{blob_id}`.
5. Metadata выполняет finalize upload и публикует объект.
6. Staging-директория удаляется.

### Multipart Upload

1. Создаётся `staging/uploads/<upload_id>/`.
2. Части пишутся как `part_00001`, `part_00002`, ...
3. На `CompleteMultipartUpload` части склеиваются в единый финальный blob.
4. Финальный blob перемещается в `blobs/{shard}/{blob_id}`.
5. После успешного finalize в Metadata staging очищается.

### DeleteObject

1. Metadata логически удаляет объект или создаёт delete marker.
2. Когда blob больше ни на что не ссылается, Metadata публикует команду удаления.
3. Storage удаляет файл сразу или через `gc/trash/`.

---

## Что Storage не должен делать

- Не строить пути по `bucket/key`
- Не считать себя источником истины о существовании объекта
- Не публиковать объект как доступный для чтения до finalize в Metadata
- Не смешивать multipart parts и финальные blob в одном каталоге

---

## Подводные камни

### 1. Staging и final blob нельзя смешивать

Если незавершённые upload лежат рядом с опубликованными blob, сложно:
- чистить мусор
- различать partially uploaded и committed data
- безопасно делать reconcile

### 2. Нужен TTL cleanup

Если клиент начал upload и исчез, в `staging/uploads/` останется мусор.
Нужен фоновый cleaner по `created_at` / `expires_at`.

### 3. Нужна атомарность publish

Переход из staging в final должен быть атомарным на уровне файловой системы:
- запись во временный файл
- `fsync`
- `rename`

### 4. Final blob лучше считать immutable

Перезапись существующего `blob_id` запрещена.
Новый PutObject должен порождать новый `blob_id`.

---

## Практический вывод

Целевая модель Storage:
- локальная FS как blob store
- staging для всех незавершённых загрузок
- immutable final blobs
- шардирование по первым символам `blob_id`
- отдельный cleanup/reconcile lifecycle

Эта схема хорошо сочетается с архитектурой `Metadata authoritative + pending/finalize`, где факт существования объекта определяется не наличием файла на диске, а успешным finalize в Metadata Service.
