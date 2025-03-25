#!/bin/bash

echo "🚀 Menghasilkan file Protobuf untuk Python..."
mkdir -p ../backend/proto_gen  # Buat folder jika belum ada
python -m grpc_tools.protoc -I=. --python_out=../backend/proto_gen --grpc_python_out=../backend/proto_gen chat.proto

echo "🚀 Menghasilkan file Protobuf untuk gRPC-Web (JavaScript)..."
mkdir -p ../frontend/src/proto_gen  # Buat folder jika belum ada
protoc -I=./proto_gen chat.proto \
  --js_out=import_style=module:../frontend/src/proto_gen \
  --grpc-web_out=import_style=module,mode=grpcwebtext:../frontend/src/proto_gen


echo "✅ Protobuf berhasil dikompilasi! 🚀"

