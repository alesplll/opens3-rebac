#!/bin/bash
set -e

REPO_ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
SHARED_PROTO="$REPO_ROOT/shared/api/authz/v1/authz.proto"
PROTO_INCLUDE="$REPO_ROOT/shared/api/authz/v1"

echo "Generating gRPC stubs from shared proto..."

mkdir -p internal/gen

python -m grpc_tools.protoc \
  -I "$PROTO_INCLUDE" \
  --python_out=internal/gen \
  --grpc_python_out=internal/gen \
  "$SHARED_PROTO"

cd internal/gen
sed -i "s/^import authz_pb2 as authz__pb2/from . import authz_pb2 as authz__pb2/" authz_pb2_grpc.py
cd ../..

touch internal/__init__.py internal/gen/__init__.py

echo "Done:"
ls -la internal/gen/
