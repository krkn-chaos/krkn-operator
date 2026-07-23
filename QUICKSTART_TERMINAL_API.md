# Quick Start - Terminal API Local Testing

## 🚀 Setup in 3 Steps

### 1. Setup gRPC Server

```bash
cd krkn-operator-data-provider

# Run setup script (one-time)
./setup-dev.sh

# Activate venv
source venv-dev/bin/activate

# Start gRPC server
python server.py
```

You should see:
```
INFO - Starting gRPC server on [::]:50051
INFO - gRPC server started successfully
```

**Keep this terminal open!**

### 2. Run Operator (new terminal)

```bash
cd krkn-operator

# Run operator locally
make run
```

You should see:
```
INFO  gRPC server address  {"address": "localhost:50051"}
INFO  🌐 Starting REST API server (waiting for JWT secret to be ready)
```

**Keep this terminal open!**

### 3. Test the API (new terminal)

First, create a test user and get a JWT token:

```bash
# Register admin user (first registered user must have role "admin")
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"userId":"admin@local.dev","password":"Admin1234!","name":"Admin","surname":"User","role":"admin"}'

# Login to get JWT token
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"userId":"admin@local.dev","password":"Admin1234!"}'

# Save the token
export TOKEN="<paste-token-here>"
```

Get a KrknTargetRequest UUID:

```bash
# List available target requests
kubectl get krkntargetrequests -n krkn-operator -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.spec.uuid}{"\n"}{end}'

# Save UUID
export UUID="<paste-uuid-here>"
```

Test terminal command:

```bash
curl -X POST http://localhost:8080/api/v1/terminal \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "cluster_id": "local-cluster",
    "uuid": "'$UUID'",
    "command": "kubectl get pods -n default"
  }' | jq
```

Expected output:

```json
{
  "stdout_base64": "TkFNRSAgICAgICAgICAgICAgICAgICAgUkVBRFkgICBTVEFUVVMgICAgUkVTVEFSVFMgICBBR0U=...",
  "stderr_base64": "",
  "exit_code": 0,
  "error": "",
  "message": ""
}
```

Decode the output:

```bash
# Decode stdout
echo "TkFNRSAgICAgICAgICAgICAgICAgICAgUkVBRFkgICBTVEFUVVMgICAgUkVTVEFSVFMgICBBR0U=" | base64 -d
```

## 🧪 Automated Testing

Use the test script:

```bash
cd krkn-operator-data-provider

# Set UUID (or will prompt)
export UUID="your-uuid-here"

# Run tests
./test-terminal-api.sh
```

## 🐛 PyCharm Setup

### Run Configuration

1. **File → Settings → Project → Python Interpreter**
   - Add interpreter → Existing → Browse to `krkn-operator-data-provider/venv-dev/bin/python`

2. **Run → Edit Configurations → + → Python**
   - Name: "gRPC Data Provider"
   - Script path: `/path/to/krkn-operator-data-provider/server.py`
   - Python interpreter: venv-dev (Python 3.11)
   - Working directory: `/path/to/krkn-operator-data-provider`

3. **Click Run** (green play button)

### Debugging

1. Set breakpoint in `server.py` (e.g., line 94 in `ExecuteKubectl`)
2. Click Debug (bug icon)
3. Make API call from terminal or frontend
4. Debugger stops at breakpoint

## 📊 What's Running?

```
┌─────────────────────────────────────────────┐
│  Terminal 1: gRPC Server (Python)          │
│  Port: localhost:50051                      │
│  Executes kubectl/oc commands               │
└─────────────────────────────────────────────┘
                    ↑
                    │ gRPC calls
                    │
┌─────────────────────────────────────────────┐
│  Terminal 2: Operator (Go)                 │
│  Port: localhost:8080 (REST API)            │
│  Handles HTTP requests, JWT auth, etc.      │
└─────────────────────────────────────────────┘
                    ↑
                    │ HTTP POST /api/v1/terminal
                    │
┌─────────────────────────────────────────────┐
│  Terminal 3 or Frontend: Client            │
│  Makes REST API calls                       │
└─────────────────────────────────────────────┘
```

## 🔥 Common Issues

### "failed to connect to gRPC server"

**Check**: gRPC server running?

```bash
lsof -i :50051
# Should show: python server.py
```

**Fix**: Start gRPC server in terminal 1

### "Command not found: kubectl"

**Check**: kubectl in PATH?

```bash
which kubectl
```

**Fix**: Install kubectl

```bash
brew install kubectl
```

### "Target request with UUID not found"

**Check**: Valid UUID?

```bash
kubectl get krkntargetrequests -n krkn-operator
```

**Fix**: Use a valid UUID from the list

### "401 Unauthorized"

**Check**: Valid JWT token?

**Fix**: Login again and update TOKEN variable

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin"}'

export TOKEN="<new-token>"
```

## 📖 More Details

- **Full setup guide**: `krkn-operator-data-provider/RUN_LOCALLY.md`
- **API documentation**: `TERMINAL_API.md`
- **Frontend tasks**: `bd list -l project:krkn-operator-console -l created-by:krkn-operator`

## ✅ Ready to Code!

Now you can:
- Modify gRPC server code and restart
- Debug with breakpoints in PyCharm
- Test changes immediately without Docker rebuild
- Iterate quickly on terminal API features
