LOCAL_BIN=$(CURDIR)/bin
VENDOR_PROTO=$(CURDIR)/shared/vendor.protogen

# ── Dependencies ───────────────────────────────────────────────────────────────

install-deps:
	GOBIN=$(LOCAL_BIN) go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28.1
	GOBIN=$(LOCAL_BIN) go install -mod=mod google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2
	GOBIN=$(LOCAL_BIN) go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@v2.20.0

vendor-proto:
	@if [ ! -d $(VENDOR_PROTO)/google ]; then \
		git clone https://github.com/googleapis/googleapis $(VENDOR_PROTO)/googleapis && \
		mkdir -p $(VENDOR_PROTO)/google/ && \
		mv $(VENDOR_PROTO)/googleapis/google/api $(VENDOR_PROTO)/google && \
		rm -rf $(VENDOR_PROTO)/googleapis ; \
	fi

# ── Proto generation ───────────────────────────────────────────────────────────

generate: generate-user generate-auth generate-storage generate-authz

generate-user:
	mkdir -p shared/pkg/user/v1
	protoc \
		--proto_path shared/api/user/v1 \
		--proto_path $(VENDOR_PROTO) \
		--go_out=shared/pkg/user/v1 --go_opt=paths=source_relative \
		--plugin=protoc-gen-go=$(LOCAL_BIN)/protoc-gen-go \
		--go-grpc_out=shared/pkg/user/v1 --go-grpc_opt=paths=source_relative \
		--plugin=protoc-gen-go-grpc=$(LOCAL_BIN)/protoc-gen-go-grpc \
		shared/api/user/v1/user.proto

generate-auth:
	mkdir -p shared/pkg/auth/v1
	protoc \
		--proto_path shared/api/auth/v1 \
		--proto_path $(VENDOR_PROTO) \
		--go_out=shared/pkg/auth/v1 --go_opt=paths=source_relative \
		--plugin=protoc-gen-go=$(LOCAL_BIN)/protoc-gen-go \
		--go-grpc_out=shared/pkg/auth/v1 --go-grpc_opt=paths=source_relative \
		--plugin=protoc-gen-go-grpc=$(LOCAL_BIN)/protoc-gen-go-grpc \
		--grpc-gateway_out=shared/pkg/auth/v1 --grpc-gateway_opt=paths=source_relative \
		--plugin=protoc-gen-grpc-gateway=$(LOCAL_BIN)/protoc-gen-grpc-gateway \
		shared/api/auth/v1/auth.proto

generate-storage:
	mkdir -p shared/pkg/storage/v1
	protoc \
		--proto_path shared/api/storage/v1 \
		--proto_path $(VENDOR_PROTO) \
		--go_out=shared/pkg/storage/v1 --go_opt=paths=source_relative \
		--plugin=protoc-gen-go=$(LOCAL_BIN)/protoc-gen-go \
		--go-grpc_out=shared/pkg/storage/v1 --go-grpc_opt=paths=source_relative \
		--plugin=protoc-gen-go-grpc=$(LOCAL_BIN)/protoc-gen-go-grpc \
		shared/api/storage/v1/storage.proto

generate-authz:
	mkdir -p shared/pkg/authz/v1
	protoc \
		--proto_path shared/api/authz/v1 \
		--proto_path $(VENDOR_PROTO) \
		--go_out=shared/pkg/authz/v1 --go_opt=paths=source_relative \
		--plugin=protoc-gen-go=$(LOCAL_BIN)/protoc-gen-go \
		--go-grpc_out=shared/pkg/authz/v1 --go-grpc_opt=paths=source_relative \
		--plugin=protoc-gen-go-grpc=$(LOCAL_BIN)/protoc-gen-go-grpc \
		shared/api/authz/v1/authz.proto

# ── Docker ─────────────────────────────────────────────────────────────────────

up-services:
	docker compose --profile services up -d

up-observability:
	docker compose --profile observability up -d

up-all:
	docker compose --profile services --profile observability up -d

down:
	docker compose down

down-volumes:
	docker compose down -v

rebuild:
	docker compose --profile services build --no-cache
	docker compose --profile services up -d
