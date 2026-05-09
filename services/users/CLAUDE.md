# CLAUDE.md — Users Service

## What this is

gRPC user management service. CRUD for user profiles + credential validation. Canonical reference for Go service patterns in this project.

Port: `:50054`

## Architecture

```
cmd/server/main.go → internal/app/app.go → internal/app/service_provider.go (DI)

internal/handler/user/
  handler.go              — implements UserServiceServer
  create.go, get.go, update.go, delete.go
  validate_credentials.go — used by Auth service for login
  update_password.go

internal/service/
  service.go              — UserService interface
  user/service.go         — implementation

internal/repository/
  repository.go           — UserRepository interface
  user/repository.go      — PostgreSQL implementation
  user/create.go, get.go, update.go, delete.go
  user/get_hashed_password.go, updatePassword.go

internal/conventer/user/user.go — domain ↔ proto conversion (note: typo in dirname is intentional)
internal/validator/              — input validation
internal/config/env/             — pg, grpc, metrics, tracing, logger, rate_limiter
```

## Key details

- User IDs are UUID strings (migrated from int64/serial)
- Migration: `migrations/20260309000000_uuid_migration.sql`
- Mocks generated via `minimock` (see `repository/generate.go`)
- This service is the **canonical reference** for handler/service/repository patterns

## Service boundaries

| Does | Does NOT |
|------|----------|
| CRUD user profiles | Issue tokens (that's Auth) |
| Validate credentials (email + password) | Authorize (that's AuthZ) |
| Store password hashes (bcrypt) | Know about S3, buckets, objects |

## Commands

```bash
go build ./services/users/...
go test ./services/users/...

# Run migrations
bash migration.sh
```
