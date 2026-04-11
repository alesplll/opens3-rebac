FROM golang:1.24.1-alpine3.21 AS builder

WORKDIR /workspace/

# Сначала копируем только go-файлы для кэширования зависимостей
COPY go.work go.work.sum ./
COPY shared/go.mod shared/go.sum ./shared/
COPY services/auth/go.mod services/auth/go.sum ./services/auth/
COPY services/metadata/go.mod services/metadata/go.sum ./services/metadata/
COPY services/storage/go.mod services/storage/go.sum ./services/storage/
COPY services/users/go.mod services/users/go.sum ./services/users/

# Скачиваем зависимости (этот слой кэшируется пока go.mod не меняются)
RUN go mod download -x 2>/dev/null; true

# Копируем весь исходный код
COPY . .

RUN go build -o ./bin/metadata-service ./services/metadata/cmd/server/main.go

FROM alpine:3.21
WORKDIR /root/
COPY --from=builder /workspace/bin/metadata-service .

CMD ["./metadata-service"]
