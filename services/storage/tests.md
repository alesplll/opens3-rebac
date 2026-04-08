# Storage Test Map

Этот файл описывает текущую структуру тестов в `services/storage`, чтобы быстро понять:
- какие слои покрыты
- где искать конкретный сценарий
- что считается unit, component и real-FS integration-style проверкой

## Слои

- `internal/handler/storage/tests` — handler unit tests. Проверяют gRPC handler-ы, чтение stream-ов и передачу данных в service.
- `internal/service/storage/tests` — service unit tests. Проверяют бизнес-валидацию и корректные вызовы repository через mocks.
- `internal/repository/storage/tests` — repository tests на реальной локальной filesystem через `t.TempDir()`. Формально это unit/integration hybrid, но по сути это main real-FS слой для storage.
- `internal/repository/storage/multipart_internal_test.go` — internal repository tests для труднодостижимых multipart edge cases через test hooks.
- `internal/tests/component` — component tests через поднятый gRPC server и реальный storage stack без внешних сервисов.

## Важно

- Отдельного integration test каталога у storage сейчас нет.
- Роль integration-style тестов для repo слоя выполняют `internal/repository/storage/tests` и `multipart_internal_test.go`, потому что они работают с реальной filesystem, а не с моками.

## Handler Unit Tests

### `internal/handler/storage/tests/store_object_test.go`

- `TestStoreObject_StreamsAllChunksToService` — проверяет, что handler склеивает чанки и передаёт их в service как единый stream.
- `TestStoreObject_EmptyStream` — проверяет ошибку на пустой client stream.

### `internal/handler/storage/tests/upload_part_test.go`

- `TestUploadPart_StreamsAllChunksToService` — проверяет потоковую передачу чанков part-а в service.
- `TestUploadPart_EmptyStream` — проверяет ошибку на пустой stream.
- `TestUploadPart_StartsStreamingBeforeClientStreamEnds` — проверяет, что service начинает читать part до завершения client stream.
- `TestUploadPart_ReturnsInvalidArgumentOnUploadIDMismatch` — проверяет ошибку при смене `upload_id` внутри одного stream-а.
- `TestUploadPart_ReturnsInvalidArgumentOnPartNumberMismatch` — проверяет ошибку при смене `part_number` внутри одного stream-а.

### `internal/handler/storage/tests/retrieve_object_test.go`

- `TestRetrieveObject_ReturnsReadError` — проверяет, что ошибка чтения из service корректно возвращается наружу.

## Service Unit Tests

### `internal/service/storage/tests/store_object_test.go`

- `TestStoreObject` — проверяет валидацию размера, нормализацию `content_type`, вызов repo и cleanup blob при size mismatch.

### `internal/service/storage/tests/retrieve_object_test.go`

- `TestRetrieveObject` — проверяет выбор между full retrieve и range retrieve и корректное прокидывание ошибок.

### `internal/service/storage/tests/delete_object_test.go`

- `TestDeleteObject` — проверяет удаление blob и прокидывание ошибок repository.

### `internal/service/storage/tests/health_check_test.go`

- `TestHealthCheck` — проверяет прокидывание результата health check из repository.

### `internal/service/storage/tests/initiate_multipart_test.go`

- `TestInitiateMultipartUpload` — проверяет создание multipart session, генерацию `upload_id` и валидацию аргументов.

### `internal/service/storage/tests/upload_part_test.go`

- `TestUploadPart` — проверяет валидацию `part_number`, передачу body в repo и прокидывание ошибок.

### `internal/service/storage/tests/complete_multipart_test.go`

- `TestCompleteMultipartUpload` — проверяет валидацию parts list, вызов `AssembleParts` и прокидывание ошибок, включая missing part.

### `internal/service/storage/tests/abort_multipart_test.go`

- `TestAbortMultipartUpload` — проверяет вызов cleanup multipart session и прокидывание ошибок.

## Repository Real-FS Tests

### `internal/repository/storage/tests/create_multipart_session_test.go`

- `TestCreateMultipartSession_Success` — проверяет создание multipart session directory в `uploads/` и `manifest.json`.

### `internal/repository/storage/tests/cleanup_multipart_test.go`

- `TestCleanupMultipart_Success` — проверяет удаление multipart session directory.
- `TestCleanupMultipart_Idempotent` — проверяет, что cleanup отсутствующей session не считается ошибкой.

### `internal/repository/storage/tests/store_part_test.go`

- `TestStorePart_Success` — проверяет успешную запись multipart part на диск.
- `TestStorePart_UploadNotFound` — проверяет ошибку при записи part в несуществующий upload.
- `TestStorePart_IgnoresStaleTempFileOnRetry` — проверяет, что stale temp не ломает retry записи part.
- `TestStorePart_RetryOverwritesExistingPart` — проверяет overwrite part при повторной загрузке того же `part_number`.
- `TestStorePart_CanceledDuringWriteCleansUpTempFile` — проверяет cleanup temp и отсутствие final part при отмене контекста во время записи.

### `internal/repository/storage/tests/store_blob_test.go`

- `TestStoreBlob_Success` — проверяет запись обычного blob на диск.
- `TestStoreBlob_CleanupTempFileOnReadError` — проверяет cleanup temp-файла при ошибке чтения из reader.
- `TestStoreBlob_LargeFile` — проверяет запись большого blob.
- `TestStoreBlob_EmptyBlob` — проверяет запись пустого blob.
- `TestStoreBlob_ContextCanceled` — проверяет поведение при уже отменённом контексте.
- `TestStoreBlob_IgnoresStaleTempFileOnRetry` — проверяет, что stale temp не ломает retry записи blob.

### `internal/repository/storage/tests/retrieve_blob_test.go`

- `TestRetrieveBlob_Success` — проверяет чтение полного blob.
- `TestRetrieveBlob_NotFound` — проверяет ошибку на отсутствующий blob.
- `TestRetrieveBlob_Range` — проверяет чтение диапазона из середины файла.
- `TestRetrieveBlob_RangePastEnd` — проверяет чтение диапазона до конца файла при слишком большом `length`.
- `TestRetrieveBlob_RangeOffsetPastEnd` — проверяет пустой результат при `offset` за концом файла.
- `TestRetrieveBlobRange_OffsetZeroFullLength` — проверяет full read через range API.

### `internal/repository/storage/tests/delete_blob_test.go`

- `TestDeleteBlob_Success` — проверяет удаление blob с диска.
- `TestDeleteBlob_Idempotent` — проверяет, что удаление отсутствующего blob не считается ошибкой.
- `TestDeleteBlob_RemovesCompletedMultipartMeta` — проверяет удаление persisted completion metadata вместе с blob.

### `internal/repository/storage/tests/health_check_test.go`

- `TestHealthCheck_DirAccessible` — проверяет успешный health check на доступной директории.
- `TestHealthCheck_DirMissing` — проверяет создание отсутствующей директории в health check flow.
- `TestHealthCheck_DataDirIsFile` — проверяет ошибку, если `DATA_DIR` указывает на файл вместо директории.

### `internal/repository/storage/tests/assemble_parts_test.go`

- `TestAssembleParts_Success` — проверяет успешную сборку multipart parts в итоговый blob.
- `TestAssembleParts_ChecksumMismatch` — проверяет ошибку при checksum mismatch части.
- `TestAssembleParts_InvalidExpectedParts` — проверяет ошибку при несовпадении `expected_parts`.
- `TestAssembleParts_SucceedsWhenCleanupFails` — проверяет best-effort cleanup после успешного commit blob.
- `TestAssembleParts_IdempotentRetryAfterCleanupFailure` — проверяет идемпотентный retry `CompleteMultipartUpload` после cleanup failure.
- `TestAssembleParts_FallsBackWhenCompletedMetaIsCorrupted` — проверяет rebuild при повреждённом completion marker.
- `TestAssembleParts_RebuildsWhenCompletedMetaExistsButBlobIsMissing` — проверяет rebuild при stale completion marker без blob.
- `TestAssembleParts_CorruptedCompletedMetaWithoutSessionReturnsUploadNotFound` — проверяет ошибку, если marker повреждён и session уже отсутствует.
- `TestAssembleParts_IgnoresStaleTempFileOnRetry` — проверяет, что stale temp не ломает retry assembly.

## Repository Internal Multipart Tests

### `internal/repository/storage/multipart_internal_test.go`

- `TestAssembleParts_CleanupUsesDetachedContext` — проверяет, что post-commit cleanup multipart session идёт через detached context и не ломается отменой request context.
- `TestAssembleParts_CanceledBeforeNextPartCopyLeavesNoBlob` — проверяет, что отмена контекста во время assembly не оставляет финальный blob.

## Component Tests

### `internal/tests/component/store_retrieve_test.go`

- `TestStoreAndRetrieveFull_SmallBlob` — проверяет полный store/retrieve маленького blob через gRPC.
- `TestStoreAndRetrieveFull_MultiChunkBlob` — проверяет многокусковый upload и full retrieve.
- `TestStoreAndRetrieveRange_MiddleSlice` — проверяет range retrieve из середины blob.
- `TestStoreAndRetrieveRange_OffsetToEnd` — проверяет range retrieve от offset до конца.
- `TestStoreAndRetrieveRange_BeyondEnd` — проверяет поведение range retrieve за концом файла.
- `TestStoreAndRetrieve_EmptyBlob` — проверяет upload и retrieve пустого blob.
- `TestRetrieve_NotFound` — проверяет `NOT_FOUND` на чтении отсутствующего blob.

### `internal/tests/component/delete_test.go`

- `TestStoreDeleteRetrieve_NotFound` — проверяет полный flow store -> delete -> not found.
- `TestDeleteNonExistent_Success` — проверяет идемпотентное удаление отсутствующего blob.

### `internal/tests/component/large_blob_test.go`

- `TestStoreAndRetrieveFull_20MB` — проверяет upload/retrieve большого blob через gRPC.

### `internal/tests/component/health_check_test.go`

- `TestHealthCheck_Serving` — проверяет успешный gRPC health check.

### `internal/tests/component/multipart_test.go`

- `TestMultipartUploadComplete_Success` — проверяет полный happy path multipart upload через gRPC.
- `TestMultipartUploadComplete_SuccessMultiChunkPart` — проверяет multipart upload, где части сами состоят из нескольких чанков.
- `TestMultipartUploadAbort_Success` — проверяет, что abort удаляет session и дальнейший `UploadPart` падает с `NOT_FOUND`.
- `TestMultipartUploadComplete_ChecksumMismatch` — проверяет `INVALID_ARGUMENT` на checksum mismatch в complete.
- `TestMultipartUploadPart_RetryOverwritesPart` — проверяет overwrite part при повторной загрузке того же `part_number`.
- `TestMultipartUploadComplete_MissingPartReturnsNotFound` — проверяет `NOT_FOUND`, если complete ссылается на отсутствующую part.
- `TestMultipartUploadAbort_AfterPartialUploadPreventsComplete` — проверяет, что после abort нельзя завершить multipart upload даже при уже записанной части.

## Infrastructure Test Files

- `internal/tests/component/main_test.go` — поднимает тестовый gRPC server и shared test environment для component tests.
- `internal/tests/component/helpers_test.go` — helper-функции для component flows (`storeBlob`, `uploadPart`, `completeMultipart` и т.д.).
- `internal/repository/storage/tests/setup_test.go` — отключает реальное логирование в repository tests.
- `internal/service/storage/tests/setup_test.go` — отключает реальное логирование в service tests.
