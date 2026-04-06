-include .env

LOCAL_BIN=$(CURDIR)/bin

install-deps:
	GOBIN=$(LOCAL_BIN) go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28.1
	GOBIN=$(LOCAL_BIN) go install -mod=mod google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2
	GOBIN=$(LOCAL_BIN) go install github.com/gojuno/minimock/v3/cmd/minimock@v3.3.5

get-deps:
	go get -u google.golang.org/protobuf/cmd/protoc-gen-go
	go get -u google.golang.org/grpc/cmd/protoc-gen-go-grpc
	go get -u google.golang.org/grpc
	go get -u github.com/grpc-ecosystem/go-grpc-middleware
	go get -u github.com/caarlos0/env/v11
	go get -u github.com/google/uuid
	go get -u github.com/joho/godotenv

generate-mocks:
	go generate ./pkg/mocks/

test:
	go test ./internal/service/... -v -count=1

rebuild:
	docker compose down
	docker compose build --no-cache
	docker compose up -d
