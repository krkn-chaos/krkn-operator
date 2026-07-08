# File Management API Specification

## Overview

The File Management API provides CRUD operations for managing configuration files stored as Kubernetes ConfigMaps. Files are automatically assigned UUIDs for identification and can be scoped to user groups or made publicly available.

## Key Concepts

### File Identification

- **Auto-Generated UUIDs**: Each file is automatically assigned a unique UUID (e.g., `550e8400-e29b-41d4-a716-446655440001`)
- **ConfigMap Naming**: ConfigMaps are named using the pattern `file-<UUID>` (e.g., `file-550e8400-e29b-41d4-a716-446655440001`)
- **Label-Based Lookup**: Files are retrieved using the `files.krkn.krkn-chaos.dev/file-id` label
- **API References**: All API endpoints use the file UUID, not the ConfigMap name

### Access Control

Files support three access modes:
1. **Group-scoped**: Accessible only to members of a specific user group (max 1 group per file)
2. **Public**: Accessible to all authenticated users (`availableToAll: true`)
3. **Private**: Not accessible to regular users (admin-only)

### Content Validation

File content must be valid JSON or YAML structured data (objects/maps or arrays/lists). Plain text files are not supported.

## API Endpoints

### Create File

**POST** `/api/v1/files`

Creates a new file and returns its auto-generated UUID.

**Request Body:**
```json
{
  "fileName": "app.conf",
  "content": "{\"server\": \"localhost\", \"port\": 8080}",
  "description": "Application configuration",
  "fileType": "config",
  "groups": ["dev-team"],
  "availableToAll": false
}
```

**Response:**
```json
{
  "message": "File created successfully",
  "fileId": "550e8400-e29b-41d4-a716-446655440001"
}
```

**Permissions:**
- Authenticated users can create public files or files for their own groups
- Admins can create files for any group

### List Files

**GET** `/api/v1/files`

Lists all files (admin only).

**Response:**
```json
{
  "files": [
    {
      "fileId": "550e8400-e29b-41d4-a716-446655440001",
      "fileName": "app.conf",
      "content": "{\"server\": \"localhost\", \"port\": 8080}",
      "description": "Application configuration",
      "fileType": "config",
      "groups": ["dev-team"],
      "availableToAll": false,
      "createdAt": "2025-01-15T10:30:00Z",
      "createdBy": "admin@example.com",
      "updatedAt": "2025-01-15T12:00:00Z",
      "updatedBy": "admin@example.com"
    }
  ],
  "total": 1
}
```

### List Available Files

**GET** `/api/v1/files/available`

Lists files accessible to the current user (based on group membership or public flag).

**Response:**
```json
{
  "files": [
    {
      "fileId": "550e8400-e29b-41d4-a716-446655440001",
      "fileName": "app.conf",
      "description": "Application configuration",
      "fileType": "config"
    }
  ]
}
```

### Get File

**GET** `/api/v1/files/{fileId}`

Retrieves a single file by UUID.

**Response:**
```json
{
  "fileId": "550e8400-e29b-41d4-a716-446655440001",
  "fileName": "app.conf",
  "content": "{\"server\": \"localhost\", \"port\": 8080}",
  "description": "Application configuration",
  "fileType": "config",
  "groups": ["dev-team"],
  "availableToAll": false,
  "createdAt": "2025-01-15T10:30:00Z",
  "createdBy": "admin@example.com"
}
```

**Permissions:**
- Users can read files they have access to (via group membership or public flag)
- Admins can read all files

### Update File

**PUT** `/api/v1/files/{fileId}`

Updates an existing file.

**Request Body:**
```json
{
  "fileName": "app.conf",
  "content": "{\"server\": \"production\", \"port\": 9090}",
  "description": "Updated application configuration",
  "fileType": "config",
  "groups": ["ops-team"],
  "availableToAll": false
}
```

**Response:**
```json
{
  "message": "File updated successfully",
  "fileId": "550e8400-e29b-41d4-a716-446655440001"
}
```

**Permissions:**
- Users can update files they have access to (via group membership)
- Admins can update all files

### Delete File

**DELETE** `/api/v1/files/{fileId}`

Deletes a file.

**Response:**
```json
{
  "message": "File deleted successfully"
}
```

**Permissions:**
- Users can delete files they have access to
- Admins can delete all files

## Kubernetes Implementation Details

### ConfigMap Structure

Files are stored as ConfigMaps with the following structure:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: file-550e8400-e29b-41d4-a716-446655440001
  namespace: krkn-operator
  labels:
    app.kubernetes.io/name: krkn-operator
    app.kubernetes.io/component: file
    files.krkn.krkn-chaos.dev/file-id: "550e8400-e29b-41d4-a716-446655440001"
    files.krkn.krkn-chaos.dev/available-to-all: "true"
    file-type.krkn.krkn-chaos.dev/config: "true"
    group.krkn.krkn-chaos.dev/dev-team: "true"
  annotations:
    files.krkn.krkn-chaos.dev/description: "Application configuration"
    files.krkn.krkn-chaos.dev/created-by: "admin@example.com"
    files.krkn.krkn-chaos.dev/created-at: "2025-01-15T10:30:00Z"
    files.krkn.krkn-chaos.dev/updated-by: "admin@example.com"
    files.krkn.krkn-chaos.dev/updated-at: "2025-01-15T12:00:00Z"
data:
  app.conf: |
    {"server": "localhost", "port": 8080}
```

### Label Naming Constraints

- **ConfigMap Name**: Max 253 characters (RFC 1123 DNS subdomain)
  - Format: `file-<UUID>` (41 characters, well within limit)
- **Label Values**: Max 63 characters
  - UUID: 36 characters (standard UUID format with hyphens)

### File Type Auto-Creation

When a file references a `fileType` that doesn't exist, a corresponding `KrknFileType` custom resource is automatically created with default settings.

## Usage Examples

### Create a Public Configuration File

```bash
curl -X POST http://localhost:8080/api/v1/files \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "fileName": "global-settings.json",
    "content": "{\"timeout\": 30, \"retries\": 3}",
    "description": "Global application settings",
    "fileType": "config",
    "availableToAll": true
  }'
```

Response:
```json
{
  "message": "File created successfully",
  "fileId": "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
}
```

### Retrieve a File by UUID

```bash
curl -X GET http://localhost:8080/api/v1/files/a1b2c3d4-e5f6-7890-abcd-ef1234567890 \
  -H "Authorization: Bearer $TOKEN"
```

### Update a File

```bash
curl -X PUT http://localhost:8080/api/v1/files/a1b2c3d4-e5f6-7890-abcd-ef1234567890 \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "fileName": "global-settings.json",
    "content": "{\"timeout\": 60, \"retries\": 5}",
    "description": "Updated global settings",
    "fileType": "config",
    "availableToAll": true
  }'
```

## Validation Rules

1. **File Content**: Must be valid JSON or YAML (objects/arrays only, no primitives)
2. **Group Assignment**: Max 1 group per file
3. **Mutual Exclusivity**: Cannot have both `groups` and `availableToAll: true`
4. **User Permissions**: Non-admin users can only assign files to their own groups or make them public
5. **Group Existence**: All referenced groups must exist as `KrknUserGroup` resources

## Error Responses

All endpoints return standard error responses:

```json
{
  "error": "error_code",
  "message": "Human-readable error message"
}
```

Common error codes:
- `unauthorized` (401): Authentication required
- `forbidden` (403): Insufficient permissions
- `not_found` (404): File not found
- `bad_request` (400): Validation error
- `internal_error` (500): Server error