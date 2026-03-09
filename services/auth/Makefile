include .env

LOCAL_BIN=$(CURDIR)/bin

install-deps:
	GOBIN=$(LOCAL_BIN) go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28.1
	GOBIN=$(LOCAL_BIN) go install -mod=mod google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2
	GOBIN=$(LOCAL_BIN) go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@v2.20.0

get-deps:
	go get -u google.golang.org/protobuf/cmd/protoc-gen-go
	go get -u google.golang.org/grpc/cmd/protoc-gen-go-grpc
	go get -u google.golang.org/grpc/
	go get -u github.com/grpc-ecosystem/go-grpc-middleware
	go get -u github.com/joho/godotenv
	go get -u github.com/WithSoull/platform_common
	go get -u github.com/WithSoull/UserServer
	go get -u github.com/golang-jwt/jwt/v5
	go get -u github.com/gomodule/redigo
	go get -u github.com/gomodule/redigo/redis
	go get -u github.com/pkg/errors
	go get -u github.com/caarlos0/env/v11
	
vendor-proto:
		@if [ ! -d vendor.protogen/google ]; then \
			git clone https://github.com/googleapis/googleapis vendor.protogen/googleapis &&\
			mkdir -p  vendor.protogen/google/ &&\
			mv vendor.protogen/googleapis/google/api vendor.protogen/google &&\
			rm -rf vendor.protogen/googleapis ;\
		fi

generate-api:
	make generate-api-auth

generate-api-auth:
	mkdir -p pkg/auth/v1
	protoc --proto_path=vendor.protogen  --proto_path api/auth/v1 \
	--go_out=pkg/auth/v1 --go_opt=paths=source_relative \
	--plugin=protoc-gen-go=bin/protoc-gen-go \
	--go-grpc_out=pkg/auth/v1 --go-grpc_opt=paths=source_relative \
	--plugin=protoc-gen-go-grpc=bin/protoc-gen-go-grpc \
	--grpc-gateway_out=pkg/auth/v1 --grpc-gateway_opt=paths=source_relative \
	--plugin=protoc-gen-grpc-gateway=bin/protoc-gen-grpc-gateway \
	api/auth/v1/auth.proto

rebuild:
	docker compose down
	docker compose build --no-cache
	docker compose up -d
