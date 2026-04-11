FROM golang:1.24.1-alpine3.21 AS builder

WORKDIR /workspace/

COPY go.work go.work.sum ./
COPY shared/go.mod ./shared/go.mod
COPY services/gateway/go.mod ./services/gateway/go.mod

RUN go mod download -x 2>/dev/null; true

COPY . .

RUN go build -o ./bin/gateway-service ./services/gateway/cmd/server/main.go

FROM alpine:3.21
WORKDIR /root/
RUN apk add --no-cache wget
COPY --from=builder /workspace/bin/gateway-service .

CMD ["./gateway-service"]
