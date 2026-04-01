include .env

LOCAL_BIN=$(CURDIR)/bin

LOCAL_MIGRATION_DIR=$(MIGRATION_DIR)
LOCAL_MIGRATION_DSN="host=${DB_HOST} port=$(PG_PORT_INNER) dbname=$(PG_DATABASE_NAME) user=$(PG_USER) password=$(PG_PASSWORD) sslmode=disable"

lint:
	$(LOCAL_BIN)/golangci-lint run ./... --config .golangci.pipeline.yaml

install-deps:
	GOBIN=$(LOCAL_BIN) go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.61.0
	GOBIN=$(LOCAL_BIN) go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28.1
	GOBIN=$(LOCAL_BIN) go install -mod=mod google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2
	GOBIN=$(LOCAL_BIN) go install github.com/pressly/goose/v3/cmd/goose@v3.14.0
	GOBIN=$(LOCAL_BIN) go install github.com/gojuno/minimock/v3/cmd/minimock@v3.3.5

get-deps:
	go get -u google.golang.org/protobuf/cmd/protoc-gen-go
	go get -u google.golang.org/protobuf/proto
	go get -u google.golang.org/grpc/cmd/protoc-gen-go-grpc
	go get -u github.com/jackc/pgx/v4
	go get -u github.com/jackc/pgconn
	go get -u google.golang.org/grpc
	go get -u golang.org/x/crypto
	go get -u github.com/gojuno/minimock/v3

	go get -u github.com/WithSoull/platform_common
	go get -u github.com/grpc-ecosystem/go-grpc-middleware
	go get -u github.com/caarlos0/env/v11


generate-api:
	mkdir -p pkg/user/v1
	protoc --proto_path api/user/v1 --proto_path vendor.protogen \
	--go_out=pkg/user/v1 --go_opt=paths=source_relative \
	--plugin=protoc-gen-go=bin/protoc-gen-go \
	--go-grpc_out=pkg/user/v1 --go-grpc_opt=paths=source_relative \
	--plugin=protoc-gen-go-grpc=bin/protoc-gen-go-grpc \
	api/user/v1/user.proto

build:
	GOOS=linux GOARCH=amd64 go build -o service_linux cmd/server/main.go

copy-to-server: build
	scp service_linux withsoull@158.160.179.174:~
	rm service_linux

# Goose (migrations)
local-migration-status:
	$(LOCAL_BIN)/goose -dir ${LOCAL_MIGRATION_DIR} postgres ${LOCAL_MIGRATION_DSN} status -v

local-migration-up:
	$(LOCAL_BIN)/goose -dir ${LOCAL_MIGRATION_DIR} postgres ${LOCAL_MIGRATION_DSN} up -v

local-migration-down:
	$(LOCAL_BIN)/goose -dir ${LOCAL_MIGRATION_DIR} postgres ${LOCAL_MIGRATION_DSN} down -v

vendor-proto:
		@if [ ! -d vendor.protogen/validate ]; then \
			mkdir -p vendor.protogen/validate &&\
			git clone https://github.com/envoyproxy/protoc-gen-validate vendor.protogen/protoc-gen-validate &&\
			mv vendor.protogen/protoc-gen-validate/validate/*.proto vendor.protogen/validate &&\
			rm -rf vendor.protogen/protoc-gen-validate ;\
		fi
		@if [ ! -d vendor.protogen/google ]; then \
			git clone https://github.com/googleapis/googleapis vendor.protogen/googleapis &&\
			mkdir -p  vendor.protogen/google/ &&\
			mv vendor.protogen/googleapis/google/api vendor.protogen/google &&\
			rm -rf vendor.protogen/googleapis ;\
		fi

gen-cert:
	openssl genrsa -out ca.key 4096
	openssl req -new -x509 -key ca.key -sha256 -subj "/C=US/ST=NJ/O=CA, Inc." -days 365 -out ca.cert
	openssl genrsa -out service.key 4096
	openssl req -new -key service.key -out service.csr -config certificate.conf
	openssl x509 -req -in service.csr -CA ca.cert -CAkey ca.key -CAcreateserial \
    		-out service.pem -days 365 -sha256 -extfile certificate.conf -extensions req_ext

rebuild:
	docker compose down
	docker compose build --no-cache
	docker compose up -d
