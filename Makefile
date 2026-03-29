LOCAL_BIN=$(CURDIR)/bin
GO_PROTO_OUT=$(CURDIR)/shared/pkg/go
PY_PROTO_OUT=$(CURDIR)/shared/pkg/py
PYTHON?=python3
PY_PROTO_DEPS=$(LOCAL_BIN)/python

# ── Dependencies ───────────────────────────────────────────────────────────────

install-deps:
	mkdir -p $(LOCAL_BIN)
	GOBIN=$(LOCAL_BIN) go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28.1
	GOBIN=$(LOCAL_BIN) go install -mod=mod google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2
	$(PYTHON) -m pip install --upgrade --target "$(PY_PROTO_DEPS)" grpcio-tools==1.74.0

# ── Proto generation ───────────────────────────────────────────────────────────

generate: generate-go generate-py

generate-all: generate
generate-all-go: generate-go
generate-all-py: generate-py

generate-go: generate-user-go generate-auth-go generate-storage-go generate-metadata-go generate-authz-go

generate-py: generate-user-py generate-auth-py generate-storage-py generate-metadata-py generate-authz-py

generate-user-go:
	mkdir -p $(GO_PROTO_OUT)/user/v1
	protoc \
		--proto_path shared/api/user/v1 \
		--go_out=$(GO_PROTO_OUT)/user/v1 --go_opt=paths=source_relative \
		--plugin=protoc-gen-go=$(LOCAL_BIN)/protoc-gen-go \
		--go-grpc_out=$(GO_PROTO_OUT)/user/v1 --go-grpc_opt=paths=source_relative \
		--plugin=protoc-gen-go-grpc=$(LOCAL_BIN)/protoc-gen-go-grpc \
		shared/api/user/v1/user.proto

generate-auth-go:
	mkdir -p $(GO_PROTO_OUT)/auth/v1
	protoc \
		--proto_path shared/api/auth/v1 \
		--go_out=$(GO_PROTO_OUT)/auth/v1 --go_opt=paths=source_relative \
		--plugin=protoc-gen-go=$(LOCAL_BIN)/protoc-gen-go \
		--go-grpc_out=$(GO_PROTO_OUT)/auth/v1 --go-grpc_opt=paths=source_relative \
		--plugin=protoc-gen-go-grpc=$(LOCAL_BIN)/protoc-gen-go-grpc \
		shared/api/auth/v1/auth.proto

generate-storage-go:
	mkdir -p $(GO_PROTO_OUT)/storage/v1
	protoc \
		--proto_path shared/api/storage/v1 \
		--go_out=$(GO_PROTO_OUT)/storage/v1 --go_opt=paths=source_relative \
		--plugin=protoc-gen-go=$(LOCAL_BIN)/protoc-gen-go \
		--go-grpc_out=$(GO_PROTO_OUT)/storage/v1 --go-grpc_opt=paths=source_relative \
		--plugin=protoc-gen-go-grpc=$(LOCAL_BIN)/protoc-gen-go-grpc \
		shared/api/storage/v1/storage.proto

generate-authz-go:
	mkdir -p $(GO_PROTO_OUT)/authz/v1
	protoc \
		--proto_path shared/api/authz/v1 \
		--go_out=$(GO_PROTO_OUT)/authz/v1 --go_opt=paths=source_relative \
		--plugin=protoc-gen-go=$(LOCAL_BIN)/protoc-gen-go \
		--go-grpc_out=$(GO_PROTO_OUT)/authz/v1 --go-grpc_opt=paths=source_relative \
		--plugin=protoc-gen-go-grpc=$(LOCAL_BIN)/protoc-gen-go-grpc \
		shared/api/authz/v1/authz.proto

generate-metadata-go:
	mkdir -p $(GO_PROTO_OUT)/metadata/v1
	protoc \
		--proto_path shared/api/metadata/v1 \
		--go_out=$(GO_PROTO_OUT)/metadata/v1 --go_opt=paths=source_relative \
		--plugin=protoc-gen-go=$(LOCAL_BIN)/protoc-gen-go \
		--go-grpc_out=$(GO_PROTO_OUT)/metadata/v1 --go-grpc_opt=paths=source_relative \
		--plugin=protoc-gen-go-grpc=$(LOCAL_BIN)/protoc-gen-go-grpc \
		shared/api/metadata/v1/metadata.proto

generate-user-py:
	mkdir -p $(PY_PROTO_OUT)/user/v1
	PYTHONPATH="$(PY_PROTO_DEPS)" $(PYTHON) -m grpc_tools.protoc \
		--proto_path=shared/api/user/v1 \
		--python_out=$(PY_PROTO_OUT)/user/v1 \
		--grpc_python_out=$(PY_PROTO_OUT)/user/v1 \
		shared/api/user/v1/user.proto
	touch shared/__init__.py shared/pkg/__init__.py $(PY_PROTO_OUT)/__init__.py $(PY_PROTO_OUT)/user/__init__.py $(PY_PROTO_OUT)/user/v1/__init__.py
	perl -0pi -e 's/^import user_pb2 as user__pb2/from . import user_pb2 as user__pb2/m' $(PY_PROTO_OUT)/user/v1/user_pb2_grpc.py

generate-auth-py:
	mkdir -p $(PY_PROTO_OUT)/auth/v1
	PYTHONPATH="$(PY_PROTO_DEPS)" $(PYTHON) -m grpc_tools.protoc \
		--proto_path=shared/api/auth/v1 \
		--python_out=$(PY_PROTO_OUT)/auth/v1 \
		--grpc_python_out=$(PY_PROTO_OUT)/auth/v1 \
		shared/api/auth/v1/auth.proto
	touch shared/__init__.py shared/pkg/__init__.py $(PY_PROTO_OUT)/__init__.py $(PY_PROTO_OUT)/auth/__init__.py $(PY_PROTO_OUT)/auth/v1/__init__.py
	perl -0pi -e 's/^import auth_pb2 as auth__pb2/from . import auth_pb2 as auth__pb2/m' $(PY_PROTO_OUT)/auth/v1/auth_pb2_grpc.py

generate-storage-py:
	mkdir -p $(PY_PROTO_OUT)/storage/v1
	PYTHONPATH="$(PY_PROTO_DEPS)" $(PYTHON) -m grpc_tools.protoc \
		--proto_path=shared/api/storage/v1 \
		--python_out=$(PY_PROTO_OUT)/storage/v1 \
		--grpc_python_out=$(PY_PROTO_OUT)/storage/v1 \
		shared/api/storage/v1/storage.proto
	touch shared/__init__.py shared/pkg/__init__.py $(PY_PROTO_OUT)/__init__.py $(PY_PROTO_OUT)/storage/__init__.py $(PY_PROTO_OUT)/storage/v1/__init__.py
	perl -0pi -e 's/^import storage_pb2 as storage__pb2/from . import storage_pb2 as storage__pb2/m' $(PY_PROTO_OUT)/storage/v1/storage_pb2_grpc.py

generate-metadata-py:
	mkdir -p $(PY_PROTO_OUT)/metadata/v1
	PYTHONPATH="$(PY_PROTO_DEPS)" $(PYTHON) -m grpc_tools.protoc \
		--proto_path=shared/api/metadata/v1 \
		--python_out=$(PY_PROTO_OUT)/metadata/v1 \
		--grpc_python_out=$(PY_PROTO_OUT)/metadata/v1 \
		shared/api/metadata/v1/metadata.proto
	touch shared/__init__.py shared/pkg/__init__.py $(PY_PROTO_OUT)/__init__.py $(PY_PROTO_OUT)/metadata/__init__.py $(PY_PROTO_OUT)/metadata/v1/__init__.py
	perl -0pi -e 's/^import metadata_pb2 as metadata__pb2/from . import metadata_pb2 as metadata__pb2/m' $(PY_PROTO_OUT)/metadata/v1/metadata_pb2_grpc.py

generate-authz-py:
	mkdir -p $(PY_PROTO_OUT)/authz/v1
	PYTHONPATH="$(PY_PROTO_DEPS)" $(PYTHON) -m grpc_tools.protoc \
		--proto_path=shared/api/authz/v1 \
		--python_out=$(PY_PROTO_OUT)/authz/v1 \
		--grpc_python_out=$(PY_PROTO_OUT)/authz/v1 \
		shared/api/authz/v1/authz.proto
	touch shared/__init__.py shared/pkg/__init__.py $(PY_PROTO_OUT)/__init__.py $(PY_PROTO_OUT)/authz/__init__.py $(PY_PROTO_OUT)/authz/v1/__init__.py
	perl -0pi -e 's/^import authz_pb2 as authz__pb2/from . import authz_pb2 as authz__pb2/m' $(PY_PROTO_OUT)/authz/v1/authz_pb2_grpc.py

# ── Tests ──────────────────────────────────────────────────────────────────────

test-users:
	go test ./services/users/... -count=1

test-users-service:
	go test ./services/users/internal/service/user/tests -count=1

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
