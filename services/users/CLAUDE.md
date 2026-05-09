# CLAUDE.md — Users Service

## What this is

gRPC user management service. CRUD for user profiles + credential validation. Canonical reference for Go service patterns in this project.

Port: `:50054`

## Architecture

```
cmd/main.go → app/app.go → service_provider.go (DI)

handler/user/
  handler.go              — implements UserServiceServer
  create.go, get.go, update.go, delete.go
  validate_credentials.go — used by Auth service for login
  update_password.go

service/
  service.go              — UserService interface

repository/user/
  repository.go           — UserRepository interface (PostgreSQL)
  create.go, get.go, update.go, delete.go
  get_hashed_password.go, updatePassword.go

conventer/user/user.go    — domain ↔ proto conversion
validator/                 — input validation
config/env/               — pg, grpc, metrics, tracing
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
