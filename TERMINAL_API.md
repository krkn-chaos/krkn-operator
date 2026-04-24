# Terminal API - Frontend Integration Guide

## Overview

The Terminal API allows frontend applications to execute read-only `kubectl` and `oc` commands against managed Kubernetes clusters with proper authentication and authorization.

**Version**: v1  
**Base Path**: `/api/v1`  
**Authentication**: JWT Bearer token (required)

## Endpoint

### Execute Terminal Command

**POST** `/api/v1/terminal`

Executes a kubectl/oc command with read-only validation.

#### Authentication

All requests must include a valid JWT token in the Authorization header:

```
Authorization: Bearer <jwt_token>
```

The token is obtained via the `/api/v1/auth/login` endpoint.

#### Request Body

```json
{
  "cluster_id": "string (required)",
  "uuid": "string (required)",
  "command": "string (required)"
}
```

**Fields:**

- `cluster_id` (string, required): The cluster name (matches the `cluster-name` field in KrknTargetRequest)
- `uuid` (string, required): The UUID of the KrknTargetRequest containing the cluster kubeconfig
- `command` (string, required): The full command string to execute (e.g., `"kubectl get pods -n default"`)

#### Response

**Success (200 OK):**

```json
{
  "stdout_base64": "string",
  "stderr_base64": "string",
  "exit_code": 0,
  "error": "",
  "message": ""
}
```

**Fields:**

- `stdout_base64` (string): Command stdout output encoded in base64
- `stderr_base64` (string): Command stderr output encoded in base64
- `exit_code` (number): Command exit code (0 = success, non-zero = error)
- `error` (string): Error type if execution failed (empty on success)
- `message` (string): Human-readable error message (empty on success)

**Error Types:**

- `not_found`: Command is not kubectl/oc or kubeconfig not found
- `not_permitted`: Command not permitted (write operations or streaming flags)
- `command_failed`: Command executed but returned non-zero exit code
- `timeout`: Command execution timed out (120s limit)
- `execution_error`: General execution error
- `invalid_command`: Failed to parse command
- `invalid_request`: Missing required fields or invalid JSON

**HTTP Status Codes:**

- `200 OK`: Command executed successfully (exit_code = 0)
- `400 Bad Request`: Command returned exit_code > 0 (includes stdout/stderr in response body)
- `403 Forbidden`: Subcommand not in whitelist or blocked flag used
- `404 Not Found`: Command is not kubectl/oc, or kubeconfig/cluster not found
- `408 Request Timeout`: Command execution exceeded 120 seconds
- `500 Internal Server Error`: Server error during execution

**Error Response (403/404/408/500):**

```json
{
  "error": "error_type",
  "message": "Human-readable error message"
}
```

**Special Case - 400 Bad Request (Command Failed):**

When a command executes but returns exit_code > 0, the response includes full output:

```json
{
  "stdout_base64": "string",
  "stderr_base64": "string",
  "exit_code": 1,
  "error": "command_failed"
}
```

## Supported Commands

### Allowed Subcommands (Read-Only)

- `get` - Get resources
- `describe` - Show details of a specific resource
- `logs` - Print container logs
- `top` - Display resource usage
- `explain` - Documentation of resources
- `version` - Print version information
- `api-resources` - Print supported API resources
- `api-versions` - Print supported API versions
- `cluster-info` - Display cluster information

### Blocked Flags (No Streaming in v1)

The following flags are blocked because they require streaming (WebSocket/SSE), which is not supported in v1:

- `--watch` / `-w` - Watch for changes
- `--follow` / `-f` - Follow log output
- `--watch-only` - Only watch for changes

## Command Parsing

Commands are parsed with support for:

- **Quoted strings**: `kubectl get pod "my pod with spaces"`
- **Long flags with equals**: `kubectl get pods --namespace=default`
- **Long flags with space**: `kubectl get pods --namespace default`
- **Short flags**: `kubectl get pods -n default`
- **Boolean flags**: `kubectl get pods --all-namespaces`

## Decoding Output

The API returns stdout and stderr as base64-encoded strings to handle binary data and special characters safely.

**JavaScript Example:**

```javascript
// Decode base64 output
const stdout = atob(response.stdout_base64);
const stderr = atob(response.stderr_base64);

console.log(stdout);
```

**TypeScript Example:**

```typescript
interface TerminalRequest {
  cluster_id: string;
  uuid: string;
  command: string;
}

interface TerminalResponse {
  stdout_base64: string;
  stderr_base64: string;
  exit_code: number;
  error?: string;
  message?: string;
}

async function executeCommand(
  clusterID: string,
  uuid: string,
  command: string,
  token: string
): Promise<{ stdout: string; stderr: string; exitCode: number }> {
  const response = await fetch('/api/v1/terminal', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${token}`
    },
    body: JSON.stringify({
      cluster_id: clusterID,
      uuid: uuid,
      command: command
    } as TerminalRequest)
  });

  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.message || 'Command execution failed');
  }

  const data: TerminalResponse = await response.json();
  
  return {
    stdout: atob(data.stdout_base64),
    stderr: atob(data.stderr_base64),
    exitCode: data.exit_code
  };
}
```

## Usage Examples

### Example 1: Get Pods

**Request:**

```json
{
  "cluster_id": "production-cluster",
  "uuid": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "command": "kubectl get pods -n default"
}
```

**Response (200 OK):**

```json
{
  "stdout_base64": "TkFNRSAgICAgICAgICAgICAgICAgICAgUkVBRFkgICBTVEFUVVMgICAgUkVTVEFSVFMgICBBR0UKbmdpbngtZGVwbG95bWVudC0xICAgICAgMi8yICAgICBSdW5uaW5nICAgMCAgICAgICAgICAxaA==",
  "stderr_base64": "",
  "exit_code": 0,
  "error": "",
  "message": ""
}
```

**Decoded stdout:**

```
NAME                     READY   STATUS    RESTARTS   AGE
nginx-deployment-1       2/2     Running   0          1h
```

### Example 2: Get Pods with Output Format

**Request:**

```json
{
  "cluster_id": "production-cluster",
  "uuid": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "command": "kubectl get pods -n default --output=yaml"
}
```

### Example 3: Describe Resource

**Request:**

```json
{
  "cluster_id": "production-cluster",
  "uuid": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "command": "kubectl describe pod nginx-deployment-1 -n default"
}
```

### Example 4: Get Logs

**Request:**

```json
{
  "cluster_id": "production-cluster",
  "uuid": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "command": "kubectl logs nginx-deployment-1 --tail=100"
}
```

### Example 5: Blocked Command (Error)

**Request:**

```json
{
  "cluster_id": "production-cluster",
  "uuid": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "command": "kubectl get pods --watch"
}
```

**Response (403 Forbidden):**

```json
{
  "error": "not_permitted",
  "message": "Command not permitted"
}
```

### Example 6: Command with Non-Zero Exit Code

**Request:**

```json
{
  "cluster_id": "production-cluster",
  "uuid": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "command": "kubectl get pod nonexistent-pod"
}
```

**Response (400 Bad Request):**

```json
{
  "stdout_base64": "",
  "stderr_base64": "RXJyb3I6IHBvZHMgIm5vbmV4aXN0ZW50LXBvZCIgbm90IGZvdW5kCg==",
  "exit_code": 1,
  "error": "command_failed"
}
```

**Decoded stderr:**

```
Error: pods "nonexistent-pod" not found
```

### Example 7: Invalid Kubeconfig (Error)

**Request:**

```json
{
  "cluster_id": "nonexistent-cluster",
  "uuid": "invalid-uuid",
  "command": "kubectl get pods"
}
```

**Response (404 Not Found):**

```json
{
  "error": "not_found",
  "message": "Failed to get kubeconfig: cluster 'nonexistent-cluster' not found in krkn-operator-acm"
}
```

## Permissions

- Users must have **view** or **edit** permission on the cluster to execute terminal commands
- Admin users have access to all clusters
- Regular users only have access to clusters shared with them via group permissions

## Timeout

- Default timeout: **120 seconds**
- Commands that exceed this timeout will return a `timeout` error
- The timeout is not configurable in v1

## Rate Limiting

- **No rate limiting** in v1
- Future versions may implement rate limiting per user/cluster

## Audit Logging

- **No audit logging** in v1
- Future versions will log all executed commands for security auditing

## Security Considerations

1. **Read-Only Operations**: Only read-only kubectl subcommands are permitted
2. **No Streaming**: Streaming commands (--watch, --follow) are blocked in v1
3. **Command Validation**: All commands are parsed and validated before execution
4. **Kubeconfig Isolation**: Each command execution uses a temporary kubeconfig file that is deleted after execution
5. **Base64 Encoding**: Output is base64-encoded to prevent XSS/injection attacks
6. **Authentication Required**: All requests require valid JWT authentication
7. **Authorization Checks**: User permissions are validated before command execution

## Future Enhancements (v2+)

The following features are not available in v1 but may be added in future versions:

- **Streaming Commands**: Support for `--watch` and `--follow` via WebSocket/SSE
- **Rate Limiting**: Per-user and per-cluster rate limits
- **Audit Logging**: Full audit trail of all executed commands
- **Output Size Limits**: Configurable limits for stdout/stderr size
- **Custom Timeout**: Per-request timeout configuration
- **Binary Detection**: Automatic detection and rejection of binary output
- **Command History**: Save and retrieve previously executed commands
- **Auto-Completion**: kubectl/oc command auto-completion suggestions

## Integration Checklist

- [ ] Implement JWT authentication flow (/api/v1/auth/login)
- [ ] Store and include JWT token in Authorization header
- [ ] Implement base64 decoding for stdout/stderr
- [ ] Handle error responses (400, 403, 404, 500)
- [ ] Display command output in terminal-like UI component
- [ ] Prevent submission of blocked flags (--watch, --follow)
- [ ] Show user-friendly error messages for validation failures
- [ ] Implement loading state during command execution (max 120s)
- [ ] Test with various kubectl commands (get, describe, logs, etc.)
- [ ] Ensure proper escaping of special characters in command input

## Support

For questions or issues with the Terminal API, contact the krkn-operator team or open an issue in the GitHub repository.
