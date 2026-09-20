# syntax=docker/dockerfile:1

FROM golang:1.24.1-alpine3.21 AS builder

WORKDIR /workspace/

# Сначала копируем только workspace manifests для кэширования зависимостей
COPY go.work go.work.sum ./
COPY e2e/go.mod ./e2e/
COPY shared/go.mod shared/go.sum ./shared/
COPY services/auth/go.mod services/auth/go.sum ./services/auth/
COPY services/gateway/go.mod services/gateway/go.sum ./services/gateway/
COPY services/metadata/go.mod services/metadata/go.sum ./services/metadata/
COPY services/storage/go.mod services/storage/go.sum ./services/storage/
COPY services/users/go.mod services/users/go.sum ./services/users/

# Ошибка скачивания должна останавливать сборку, а успешно загруженные модули
# сохраняются в BuildKit cache и переиспользуются при повторном запуске.
RUN --mount=type=cache,target=/go/pkg/mod \
    cd services/storage && GOWORK=off go mod download

# Копируем весь исходный код
COPY . .

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -o ./bin/storage-service ./services/storage/cmd/server/main.go

FROM alpine:3.21
WORKDIR /root/
COPY --from=builder /workspace/bin/storage-service .

CMD ["./storage-service"]
