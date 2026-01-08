#!/bin/bash
# generate_proto.sh
# Script to generate Python gRPC code from protobuf definitions

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROTO_DIR="$SCRIPT_DIR/../proto"
OUT_DIR="$SCRIPT_DIR/generated"

echo "Generating Python gRPC code..."
echo "Proto directory: $PROTO_DIR"
echo "Output directory: $OUT_DIR"

# Create output directory if it doesn't exist
mkdir -p "$OUT_DIR"

# Generate Python code
python3 -m grpc_tools.protoc \
    -I"$PROTO_DIR" \
    --python_out="$OUT_DIR" \
    --pyi_out="$OUT_DIR" \
    --grpc_python_out="$OUT_DIR" \
    "$PROTO_DIR/dataprovider.proto"

# Fix imports in generated files to use relative imports
sed -i '' 's/^import dataprovider_pb2/from . import dataprovider_pb2/g' "$OUT_DIR/dataprovider_pb2_grpc.py"

# Create __init__.py to make it a package
touch "$OUT_DIR/__init__.py"

echo "Python gRPC code generated successfully in $OUT_DIR"