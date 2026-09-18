FROM golang:1.24.1-alpine3.21 AS builder

WORKDIR /workspace
COPY . .
RUN CGO_ENABLED=0 go build -o /gateway ./services/gateway/cmd/server

FROM alpine:3.21
COPY --from=builder /gateway /gateway
USER 65532:65532
EXPOSE 8080
CMD ["/gateway"]
