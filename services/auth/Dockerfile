FROM golang:1.24.1-alpine3.21 AS builder

COPY . /workspace/
WORKDIR /workspace/

RUN go work sync
RUN go build -o ./bin/auth-service ./services/auth/cmd/server/main.go

FROM alpine:3.21
WORKDIR /root/
COPY --from=builder /workspace/bin/auth-service .
COPY services/auth/.env .

CMD ["./auth-service", "-config-path=/root/.env"]
