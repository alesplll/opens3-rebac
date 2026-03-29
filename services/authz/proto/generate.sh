#!/bin/bash
set -e

REPO_ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
SHARED_PROTO="$REPO_ROOT/shared/api/authz/v1/authz.proto"
PROTO_INCLUDE="$REPO_ROOT/shared/api/authz/v1"
OUT_DIR="$REPO_ROOT/shared/pkg/py/authz/v1"
PYTHON_BIN="${PYTHON_BIN:-python3}"
PY_PROTO_DEPS="${PY_PROTO_DEPS:-$REPO_ROOT/bin/python}"

echo "Generating gRPC stubs from shared proto..."

mkdir -p "$OUT_DIR"

PYTHONPATH="$PY_PROTO_DEPS" "$PYTHON_BIN" -m grpc_tools.protoc \
  -I "$PROTO_INCLUDE" \
  --python_out="$OUT_DIR" \
  --grpc_python_out="$OUT_DIR" \
  "$SHARED_PROTO"

perl -0pi -e 's/^import authz_pb2 as authz__pb2/from . import authz_pb2 as authz__pb2/m' "$OUT_DIR/authz_pb2_grpc.py"

touch "$REPO_ROOT/shared/__init__.py" \
  "$REPO_ROOT/shared/pkg/__init__.py" \
  "$REPO_ROOT/shared/pkg/py/__init__.py" \
  "$REPO_ROOT/shared/pkg/py/authz/__init__.py" \
  "$REPO_ROOT/shared/pkg/py/authz/v1/__init__.py"

echo "Done:"
ls -la "$OUT_DIR"
