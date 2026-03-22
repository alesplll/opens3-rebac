FROM golang:1.24.1-alpine3.21 AS builder

WORKDIR /workspace/

# Сначала копируем только go-файлы для кэширования зависимостей
COPY go.work go.work.sum ./
COPY shared/go.mod shared/go.sum ./shared/
COPY services/users/go.mod services/users/go.sum ./services/users/
COPY services/auth/go.mod services/auth/go.sum ./services/auth/

# Скачиваем зависимости (этот слой кэшируется пока go.mod не меняются)
RUN go mod download -x 2>/dev/null; true

# Копируем весь исходный код
COPY . .

RUN go build -o ./bin/auth-service ./services/auth/cmd/server/main.go

FROM alpine:3.21
WORKDIR /root/
COPY --from=builder /workspace/bin/auth-service .

CMD ["./auth-service"]
