# Storage filesystem architecture

Статус: **текущее устройство single-node Storage и открытые ограничения**.
Обновлено: 2026-09-11.

Storage работает с `blob_id`, `upload_id` и multipart parts. Он не знает S3 bucket,
key или permission. Metadata остаётся catalog authority.

## Layout

Фактические пути строятся конфигурацией repository и шардируются по первым
символам UUID. Концептуально:

```text
DATA_DIR/<shard>/<blob_id>                         final blobs
MULTIPART_DIR/uploads/<upload_id>/                 session and parts
MULTIPART_DIR/completed/<shard>/<upload_id>.json   completion metadata
```

Single-part upload также использует temporary/staging file до final rename.
`DATA_DIR` и staging path должны находиться на одной filesystem, иначе rename не
имеет ожидаемой атомарности.

## Single-part write

1. Service создаёт blob UUID до repository write.
2. Repository создаёт temporary file.
3. Входной stream копируется в файл с подсчётом размера и MD5.
4. Файл синхронизируется и закрывается.
5. Проверяется ожидаемый размер, если он передан.
6. Temporary file переименовывается в final path.
7. Service возвращает blob ID и full-content MD5.

Final path появляется только после rename. Пустой blob допустим.

`fsync(file)` плюс rename обеспечивает атомарную видимость имени на одной
filesystem, но текущий код не делает `fsync` каталога. Поэтому документ не обещает,
что новое имя гарантированно переживёт power loss во всех filesystems.

## Multipart

Initiate создаёт session и upload ID. UploadPart сохраняет либо заменяет part по
номеру. Complete принимает упорядоченный список `(part_number, checksum_md5)`,
проверяет существование/checksum и собирает итоговый blob. В текущей реализации
его blob ID совпадает с upload ID.

После commit Storage сохраняет completion metadata, затем best-effort удаляет
session. Cleanup не ждёт Metadata finalize. Completion metadata нужна, чтобы retry
Complete после потерянного ответа мог вернуть тот же результат.

Issue #36 может менять срок хранения marker только вместе с durable terminal
result в Metadata или другом registry. Иначе клиентский retry после успешного
finalize и потерянного ответа станет неразрешимым.

Внутренний MD5 итогового файла не является универсальным S3 multipart ETag.
Внешний Gateway обязан реализовать выбранный checksum mode отдельно.

## Видимость Metadata

Данные становятся final внутри Storage после успешного Complete/StoreObject.
Будущий внешний PUT считается успешным только после регистрации committed version
в Metadata. Если будет выбран pending/finalize protocol, он должен включать
durable intent, точный version/operation ID, storage acknowledgement и атомарную
смену current pointer.

Pending overwrite не скрывает прежнюю committed current version. Новый key без
current version остаётся невидимым до commit.

## Cleanup

- abort удаляет multipart session;
- stale sessions удаляются отдельной политикой по возрасту;
- orphan final blobs требуют сверки с durable Metadata intent/event;
- delete marker не удаляет historical version blobs;
- permanent delete/GC должны адресовать конкретные blob/version IDs.

Текущий сервис не содержит Kafka cleanup consumer. Topic names и event schemas
нужно согласовать до реализации; `object-stored`, `object-deleted` и abort events
не считаются готовым runtime только из-за описания в proto/comments.

## Health

Custom `DataStorageService.HealthCheck` проверяет filesystem repository. Standard
`grpc.health.v1.Health` получает SERVING при старте и не эквивалентен динамической
проверке каталога.

## Требуемые тесты

- empty/large/cancelled stream и size mismatch;
- cleanup temporary file после read/write error;
- retry StoreObject/UploadPart/Complete;
- atomic visibility до/после rename;
- session cleanup failure после успешного commit;
- power-loss tests для выбранной filesystem/durability policy;
- orphan reconciliation и version-aware deletion после появления событий.
