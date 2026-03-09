FROM golang:1.24.1-alpine3.21 AS builder

COPY . /workspace/
WORKDIR /workspace/

RUN go work sync
RUN go build -o ./bin/users-service ./services/users/cmd/server/main.go

FROM alpine:3.21
WORKDIR /root/
COPY --from=builder /workspace/bin/users-service .
COPY services/users/.env .

CMD ["./users-service", "-config-path=/root/.env"]
