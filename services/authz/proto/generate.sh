#!/bin/bash
set -e

echo "🛠️ Generating gRPC stubs..."

mkdir -p internal/gen

python -m grpc_tools.protoc \
  -I proto \
  --python_out=internal/gen \
  --grpc_python_out=internal/gen \
  proto/authz.proto

cd internal/gen
sed -i "s/^import authz_pb2 as authz__pb2/from . import authz_pb2 as authz__pb2/" authz_pb2_grpc.py
sed -i "s/^import authz_pb2_grpc as grpc/from . import authz_pb2_grpc as grpc/" authz_pb2_grpc.py 2>/dev/null || true
cd ../..

touch internal/__init__.py internal/gen/__init__.py

echo "✅ Fixed imports:"
ls -la internal/gen/
echo "✅ Ready to run!"
