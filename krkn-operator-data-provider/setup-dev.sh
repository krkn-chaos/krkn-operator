#!/bin/bash
# setup-dev.sh
# Quick setup script for local development

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "🔧 Setting up krkn-operator-data-provider for local development..."
echo ""

# Check Python 3.11
if ! command -v python3.11 &> /dev/null; then
    echo "❌ Error: python3.11 not found"
    echo "   Install with: brew install python@3.11"
    exit 1
fi

echo "✅ Found Python 3.11: $(python3.11 --version)"

# Create venv if not exists
if [ ! -d "venv-dev" ]; then
    echo "📦 Creating virtual environment (venv-dev)..."
    python3.11 -m venv venv-dev
else
    echo "✅ Virtual environment already exists"
fi

# Activate venv
echo "🔌 Activating virtual environment..."
source venv-dev/bin/activate

# Upgrade pip
echo "⬆️  Upgrading pip..."
pip install --upgrade pip -q

# Install dependencies
echo "📥 Installing dependencies..."
pip install grpcio>=1.60.0 grpcio-tools>=1.60.0 -q

echo "📥 Installing krkn-lib (this may take a moment)..."
pip install git+https://github.com/krkn-chaos/krkn-lib.git@init_from_string -q

# Verify generated stubs exist
if [ ! -f "generated/dataprovider_pb2.py" ]; then
    echo "⚠️  Generated stubs not found, regenerating..."
    python3.11 -m grpc_tools.protoc \
        -I../proto \
        --python_out=generated \
        --pyi_out=generated \
        --grpc_python_out=generated \
        ../proto/dataprovider.proto

    # Fix imports
    sed -i '' 's/^import dataprovider_pb2/from . import dataprovider_pb2/g' generated/dataprovider_pb2_grpc.py
    touch generated/__init__.py
    echo "✅ Generated Python stubs"
else
    echo "✅ Generated stubs already exist"
fi

# Check kubectl/oc
echo ""
echo "🔍 Checking kubectl/oc availability..."
if command -v kubectl &> /dev/null; then
    echo "✅ kubectl found: $(which kubectl)"
else
    echo "⚠️  kubectl not found - install with: brew install kubectl"
fi

if command -v oc &> /dev/null; then
    echo "✅ oc found: $(which oc)"
else
    echo "⚠️  oc not found (optional) - download from OpenShift CLI"
fi

echo ""
echo "✅ Setup complete!"
echo ""
echo "📝 Next steps:"
echo "   1. Activate venv: source venv-dev/bin/activate"
echo "   2. Start server:  python server.py"
echo "   3. In another terminal, run operator: make run"
echo ""
echo "📖 See RUN_LOCALLY.md for detailed instructions"
