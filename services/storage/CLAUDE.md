# CLAUDE.md — Storage Service

## What this is

gRPC blob storage service. Stores and retrieves raw bytes on the local filesystem. Supports simple put/get/delete and multipart uploads.

Port: `:50053`

## Architecture

```
cmd/server/main.go → internal/app/app.go → internal/app/service_provider.go (DI)

internal/handler/storage/
  handler.go           — implements StorageServiceServer
  store_object.go      — StoreObject: client-streaming (chunks → blob)
  retrieve_object.go   — RetrieveObject: server-streaming (blob → chunks)
  delete_object.go     — DeleteObject
  initiate_multipart.go
  upload_part.go
  complete_multipart.go
  abort_multipart.go
  health_check.go
  chunk_reader.go      — helper: reads io.Reader in fixed-size chunks
  upload_streams.go    — stream adapter helpers
  client_stream.go

internal/service/storage/
  service.go           — StorageService interface + constructor
  store_object.go
  retrieve_object.go
  delete_object.go
  initiate_multipart.go
  upload_part.go
  complete_multipart.go
  abort_multipart.go
  health_check.go

internal/repository/
  repository.go              — StorageRepository interface
  storage/repository.go      — filesystem implementation
  storage/store_blob.go, retrieve_blob.go, delete_blob.go
  storage/multipart.go       — multipart session management
  storage/health.go
```

## Key details

- Blobs stored by `blob_id` (UUID), not by object key — mapping is in Metadata service
- Streaming: StoreObject uses client-streaming, RetrieveObject uses server-streaming
- Multipart: initiate → upload parts → complete/abort

## Service boundaries

| Does | Does NOT |
|------|----------|
| Store/retrieve/delete raw bytes | Know about object keys or metadata |
| Generate blob_ids | Check permissions |
| Support multipart uploads | Know about S3 API |

## Commands

```bash
go build ./services/storage/...
go test ./services/storage/...
```
