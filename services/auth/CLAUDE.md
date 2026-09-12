# CLAUDE.md — Auth Service

## What this is

gRPC authentication service. Issues and validates JWT tokens (access + refresh). Delegates credential verification to the Users service via gRPC client.

Port: `:50050`

## Architecture

```
cmd/server/main.go → internal/app/app.go → internal/app/service_provider.go (DI)

internal/handler/auth/
  handler.go           — implements AuthServiceServer
  login.go             — Login RPC: validate credentials via Users client → issue tokens
  get_access_token.go  — reissue access token from refresh token
  get_refresh_token.go — reissue refresh token
  validate_token.go    — ValidateToken RPC: verify JWT, return claims

internal/service/
  service.go           — AuthService interface
  auth/
    service.go         — orchestrates UserClient + TokenService + Repository
    login.go
    verify_token.go
    get_access_token.go
    get_refresh_token.go

internal/repository/auth/ — refresh token persistence (Redis)
internal/client/grpc/     — gRPC client to Users service
internal/client/cache/    — Redis cache client
internal/config/env/      — env var configs (jwt, redis, grpc, security, rate_limiter, metrics, tracing, logger, user_grpc)
```

## Key dependencies

- `shared/pkg/go-kit/tokens` — JWT token creation/validation (shared lib)
- Users service (`:50054`) — credential verification via gRPC

## Service boundaries

| Does | Does NOT |
|------|----------|
| Issue JWT (access + refresh) | Authorize (that's AuthZ) |
| Validate tokens | Manage user profiles |
| Store refresh tokens (Redis) | Know about S3, buckets, objects |

## Commands

```bash
go build ./services/auth/...
go test ./services/auth/...
```
