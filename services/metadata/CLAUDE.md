# CLAUDE.md — Metadata Service

## What this is

gRPC metadata service. Stores object/bucket metadata in PostgreSQL. Publishes events to Kafka when objects/buckets are created or deleted.

Port: `:50052`

## Architecture

```
cmd/main.go → (standard app/service_provider pattern)

handler/metadata/
  handler.go — implements MetadataServiceServer (currently skeleton)
```

**Status:** skeleton — handler embeds `UnimplementedMetadataServiceServer`. Business logic not yet implemented.

## Planned responsibilities

- CRUD for bucket metadata (name, owner, created_at)
- CRUD for object metadata (key, bucket, size, content_type, blob_id, version)
- Kafka producer: `object-stored`, `object-deleted`, `bucket-deleted`
- Kafka consumer: none (receives calls from Gateway)

## Service boundaries

| Does | Does NOT |
|------|----------|
| Store object/bucket metadata (PostgreSQL) | Store bytes (that's Storage) |
| Emit Kafka events on mutations | Check permissions (that's AuthZ) |
| Map object keys to blob_ids | Know about S3 API directly |

## Commands

```bash
go build ./services/metadata/...
go test ./services/metadata/...
```
