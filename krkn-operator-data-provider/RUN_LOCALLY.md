# Running gRPC Data Provider Locally for Development

## Prerequisites

- Python 3.11+ (required by krkn-lib)
- kubectl or oc CLI installed and in PATH
- krkn-operator running locally with `make run`

## Setup

### 1. Create Virtual Environment (Python 3.11)

```bash
cd krkn-operator-data-provider

# Create venv with Python 3.11
python3.11 -m venv venv-dev

# Activate venv
source venv-dev/bin/activate

# Upgrade pip
pip install --upgrade pip
```

### 2. Install Dependencies

```bash
# Install dependencies
pip install grpcio>=1.60.0 grpcio-tools>=1.60.0

# Install krkn-lib (requires Python 3.11+)
pip install git+https://github.com/krkn-chaos/krkn-lib.git@init_from_string
```

### 3. Verify Generated Stubs

The Python stubs should already be generated in `generated/`:

```bash
ls -la generated/
# Should see: dataprovider_pb2.py, dataprovider_pb2.pyi, dataprovider_pb2_grpc.py
```

If missing, regenerate:

```bash
python3.11 -m grpc_tools.protoc \
  -I../proto \
  --python_out=generated \
  --pyi_out=generated \
  --grpc_python_out=generated \
  ../proto/dataprovider.proto

# Fix imports
sed -i '' 's/^import dataprovider_pb2/from . import dataprovider_pb2/g' generated/dataprovider_pb2_grpc.py
```

### 4. Start gRPC Server

```bash
# Make sure venv is activated
source venv-dev/bin/activate

# Start server (listens on localhost:50051)
python server.py
```

You should see:

```
INFO - Starting gRPC server on [::]:50051
INFO - gRPC server started successfully
```

## Configure Operator to Use Local gRPC Server

### Option A: Environment Variable (Recommended)

Edit your shell or IDE run configuration to set:

```bash
export GRPC_SERVER_ADDR=localhost:50051
```

Then run operator:

```bash
make run
```

### Option B: Modify main.go Temporarily

**⚠️ DO NOT COMMIT THIS CHANGE**

Edit `cmd/main.go`:

```go
// Find this line (around line 150-160):
grpcServerAddr := os.Getenv("GRPC_SERVER_ADDR")
if grpcServerAddr == "" {
    grpcServerAddr = "krkn-operator-data-provider:50051" // Change this to "localhost:50051"
}
```

Change to:

```go
grpcServerAddr := os.Getenv("GRPC_SERVER_ADDR")
if grpcServerAddr == "" {
    grpcServerAddr = "localhost:50051" // <-- LOCAL FOR TESTING
}
```

Then:

```bash
make run
```

**Remember to revert before committing!**

### Option C: Port Forward (if operator runs in cluster)

If your operator is running **in the cluster** (not locally), you need to expose the local gRPC server:

```bash
# Terminal 1: Start gRPC server locally
cd krkn-operator-data-provider
source venv-dev/bin/activate
python server.py

# Terminal 2: Port forward from cluster to local
kubectl port-forward -n krkn-operator deployment/krkn-operator-manager 50051:50051

# This won't work as expected - better to run operator locally
```

**Recommendation**: Run operator locally with `make run` for easier development.

## Testing the Terminal API

### 1. Login and Get JWT Token

```bash
# Register/Login to get JWT token
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"yourpassword"}'

# Save token
export TOKEN="<jwt_token_from_response>"
```

### 2. Create KrknTargetRequest (or use existing UUID)

You need a valid KrknTargetRequest with cluster kubeconfig. Check existing ones:

```bash
kubectl get krkntargetrequests -n krkn-operator

# Get UUID from one
kubectl get krkntargetrequest <name> -n krkn-operator -o jsonpath='{.spec.uuid}'
```

### 3. Test Terminal Command

```bash
# Example: Get pods in default namespace
curl -X POST http://localhost:8080/api/v1/terminal \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "cluster_id": "local-cluster",
    "uuid": "<your-uuid-here>",
    "command": "kubectl get pods -n default"
  }'
```

Expected response:

```json
{
  "stdout_base64": "TkFNRSAgICAgICAgICAgICAgICAgICAgUkVBRFkgICBTVEFUVVMgICAgUkVTVEFSVFMgICBBR0U=...",
  "stderr_base64": "",
  "exit_code": 0,
  "error": "",
  "message": ""
}
```

Decode output:

```bash
echo "TkFNRSAgICAgICAgICAgICAgICAgICAgUkVBRFkgICBTVEFUVVMgICAgUkVTVEFSVFMgICBBR0U=..." | base64 -d
```

## Troubleshooting

### Error: "no module named 'krkn_lib'"

```bash
# Make sure venv-dev is activated
source venv-dev/bin/activate

# Reinstall krkn-lib
pip install git+https://github.com/krkn-chaos/krkn-lib.git@init_from_string
```

### Error: "Command not found: kubectl"

The gRPC server executes kubectl/oc via subprocess. Make sure they're in your PATH:

```bash
which kubectl
# Should show: /usr/local/bin/kubectl (or similar)

which oc
# Should show: /usr/local/bin/oc (or similar)
```

If not installed:

```bash
# macOS
brew install kubectl

# or download oc CLI from OpenShift
```

### Error: "failed to connect to gRPC server"

Check that gRPC server is running:

```bash
# Should show: python server.py listening on port 50051
lsof -i :50051
```

Check operator logs:

```bash
# If running with make run, check terminal output
# Look for: "Failed to connect to gRPC server: localhost:50051"
```

### Error: "Failed to get kubeconfig"

Make sure you have a valid KrknTargetRequest with kubeconfig secret:

```bash
# List target requests
kubectl get krkntargetrequests -n krkn-operator

# Check secret exists
kubectl get secret <uuid> -n krkn-operator
```

## PyCharm Configuration

### Run Configuration for gRPC Server

1. **Create new Python configuration**:
   - Script path: `/path/to/krkn-operator-data-provider/server.py`
   - Python interpreter: venv-dev (Python 3.11)
   - Working directory: `/path/to/krkn-operator-data-provider`

2. **Environment variables** (optional):
   - `GRPC_PORT=50051`
   - `LOG_LEVEL=DEBUG` (for verbose logging)

3. **Run**:
   - Click Run → Should see "Starting gRPC server on [::]:50051"

### Debug gRPC Server in PyCharm

1. Set breakpoints in `server.py` (e.g., in `ExecuteKubectl` method)
2. Click Debug instead of Run
3. Make API call from frontend or curl
4. Debugger will stop at breakpoint

## Development Workflow

```bash
# Terminal 1: gRPC Server (PyCharm or command line)
cd krkn-operator-data-provider
source venv-dev/bin/activate
python server.py

# Terminal 2: Operator
cd krkn-operator
export GRPC_SERVER_ADDR=localhost:50051
make run

# Terminal 3: Frontend (if needed)
cd krkn-operator-console
npm run dev

# Terminal 4: Test API calls
curl -X POST http://localhost:8080/api/v1/terminal ...
```

## Cleanup

When done testing:

```bash
# Stop gRPC server (Ctrl+C)
# Stop operator (Ctrl+C)
# Deactivate venv
deactivate

# Revert any changes to main.go if you modified it
git checkout cmd/main.go
```

## Notes

- **Do not commit** venv-dev/ (already in .gitignore)
- **Do not commit** temporary changes to main.go for GRPC_SERVER_ADDR
- **Use environment variable** for configuration instead of code changes
- **kubectl/oc must be in PATH** for command execution to work
- **Python 3.11+ required** for krkn-lib compatibility
