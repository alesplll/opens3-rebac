# syntax=docker/dockerfile:1

FROM golang:1.24.1-alpine3.21 AS builder

WORKDIR /workspace

COPY go.work go.work.sum ./
COPY e2e/go.mod ./e2e/
COPY shared/go.mod shared/go.sum ./shared/
COPY services/auth/go.mod services/auth/go.sum ./services/auth/
COPY services/gateway/go.mod services/gateway/go.sum ./services/gateway/
COPY services/metadata/go.mod services/metadata/go.sum ./services/metadata/
COPY services/storage/go.mod services/storage/go.sum ./services/storage/
COPY services/users/go.mod services/users/go.sum ./services/users/

RUN --mount=type=cache,target=/go/pkg/mod \
    cd services/gateway && GOWORK=off go mod download

COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -o /gateway ./services/gateway/cmd/server

FROM alpine:3.21
COPY --from=builder /gateway /gateway
USER 65532:65532
EXPOSE 8080
CMD ["/gateway"]
