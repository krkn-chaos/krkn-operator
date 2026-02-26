# Authentication API Specification

## Overview

The krkn-operator REST API implements JWT-based authentication with role-based authorization (admin/user).

**Base URL**: `http://<operator-host>:<port>/api/v1`

**Authentication Flow**:
1. Check if admin is registered using `/auth/is-registered`
2. Register first admin (no auth required) or login
3. Use JWT token in `Authorization` header for all subsequent requests

---

## Endpoints

### 1. Check Admin Registration

Check if at least one admin user exists in the system.

**Endpoint**: `GET /auth/is-registered`

**Authentication**: None (public endpoint)

**Request**: No body

**Response** (200 OK):
```json
{
  "registered": true
}
```

**Response Fields**:
- `registered` (boolean): `true` if at least one admin exists, `false` otherwise

**Example**:
```bash
curl -X GET http://localhost:8080/api/v1/auth/is-registered
```

---

### 2. Register User

Register a new user in the system.

**Endpoint**: `POST /auth/register`

**Authentication**:
- **First admin**: No authentication required
- **Subsequent users**: Requires admin JWT token (not yet enforced - planned for future)

**Request Body**:
```json
{
  "userId": "[email protected]",
  "password": "SecurePassword123",
  "name": "John",
  "surname": "Doe",
  "organization": "Example Corp",
  "role": "admin"
}
```

**Request Fields**:
- `userId` (string, required): User's email address (must be valid email format)
- `password` (string, required): Password (minimum 8 characters)
- `name` (string, required): User's first name
- `surname` (string, required): User's last name
- `organization` (string, optional): User's organization
- `role` (string, required): Either `"user"` or `"admin"`

**Response** (201 Created):
```json
{
  "message": "User registered successfully",
  "userId": "[email protected]",
  "role": "admin"
}
```

**Response Fields**:
- `message` (string): Success message
- `userId` (string): Registered user's email
- `role` (string): User's role

**Error Responses**:

**400 Bad Request** - Validation error:
```json
{
  "error": "validation_error",
  "message": "Password must be at least 8 characters"
}
```

**400 Bad Request** - First user must be admin:
```json
{
  "error": "validation_error",
  "message": "First user must have admin role"
}
```

**409 Conflict** - User already exists:
```json
{
  "error": "user_exists",
  "message": "User with email [email protected] already exists"
}
```

**Example**:
```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "userId": "[email protected]",
    "password": "SecurePassword123",
    "name": "John",
    "surname": "Doe",
    "organization": "Example Corp",
    "role": "admin"
  }'
```

---

### 3. Login

Authenticate a user and receive a JWT token.

**Endpoint**: `POST /auth/login`

**Authentication**: None (public endpoint)

**Request Body**:
```json
{
  "userId": "[email protected]",
  "password": "SecurePassword123"
}
```

**Request Fields**:
- `userId` (string, required): User's email address
- `password` (string, required): User's password

**Response** (200 OK):
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expiresAt": "2026-02-27T10:30:00Z",
  "userId": "[email protected]",
  "role": "admin",
  "name": "John",
  "surname": "Doe"
}
```

**Response Fields**:
- `token` (string): JWT authentication token (valid for 24 hours)
- `expiresAt` (string, RFC3339): Token expiration timestamp
- `userId` (string): Authenticated user's email
- `role` (string): User's role (`"user"` or `"admin"`)
- `name` (string): User's first name
- `surname` (string): User's last name

**Error Responses**:

**401 Unauthorized** - Invalid credentials:
```json
{
  "error": "invalid_credentials",
  "message": "Invalid email or password"
}
```

**401 Unauthorized** - Account disabled:
```json
{
  "error": "account_disabled",
  "message": "User account is disabled"
}
```

**Example**:
```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "userId": "[email protected]",
    "password": "SecurePassword123"
  }'
```

---

## Using JWT Token

After successful login, include the JWT token in the `Authorization` header for all authenticated requests:

**Header Format**:
```
Authorization: Bearer <token>
```

**Example**:
```bash
curl -X GET http://localhost:8080/api/v1/clusters \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

**Authentication Errors**:

**401 Unauthorized** - Missing token:
```json
{
  "error": "unauthorized",
  "message": "Missing authorization token"
}
```

**401 Unauthorized** - Invalid token format:
```json
{
  "error": "unauthorized",
  "message": "Invalid authorization header format. Expected: Bearer <token>"
}
```

**401 Unauthorized** - Invalid or expired token:
```json
{
  "error": "unauthorized",
  "message": "Invalid or expired token"
}
```

**403 Forbidden** - Insufficient permissions:
```json
{
  "error": "forbidden",
  "message": "This operation requires admin privileges"
}
```

---

## Authorization Rules

### Public Endpoints (No Authentication)
- `GET /auth/is-registered`
- `POST /auth/register`
- `POST /auth/login`

### Authenticated Endpoints (User + Admin)
All other endpoints require authentication. Include JWT token in `Authorization` header.

**User and Admin access**:
- `GET /health`, `GET /clusters`, `GET /nodes`
- `POST/GET /targets` (legacy endpoints)
- All scenario endpoints: `POST /scenarios`, `POST /scenarios/detail/*`, etc.
- `GET /operator/targets`, `GET /operator/targets/{uuid}`
- `GET /provider-config/{uuid}`
- `GET /providers`, `GET /providers/{name}`

### Admin-Only Operations
These endpoints/methods require admin role:

- `POST /operator/targets` - Create new target
- `PUT /operator/targets/{uuid}` - Update target
- `DELETE /operator/targets/{uuid}` - Delete target
- `POST /provider-config` - Create provider config
- `POST /provider-config/{uuid}` - Update provider config
- `PATCH /providers/{name}` - Update provider status

---

## Frontend Implementation Flow

### 1. Initial Load
```javascript
// Check if admin is registered
const response = await fetch('/api/v1/auth/is-registered');
const { registered } = await response.json();

if (!registered) {
  // Show "First Admin Registration" page
} else {
  // Show login page
}
```

### 2. First Admin Registration
```javascript
const registerData = {
  userId: "[email protected]",
  password: "SecurePassword123",
  name: "John",
  surname: "Doe",
  organization: "Example Corp",
  role: "admin"
};

const response = await fetch('/api/v1/auth/register', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify(registerData)
});

if (response.ok) {
  // Registration successful, redirect to login
} else {
  const error = await response.json();
  // Show error message
}
```

### 3. Login
```javascript
const loginData = {
  userId: "[email protected]",
  password: "SecurePassword123"
};

const response = await fetch('/api/v1/auth/login', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify(loginData)
});

if (response.ok) {
  const data = await response.json();

  // Store token (localStorage or sessionStorage)
  localStorage.setItem('jwt_token', data.token);
  localStorage.setItem('user_role', data.role);
  localStorage.setItem('user_name', data.name);
  localStorage.setItem('user_email', data.userId);

  // Redirect to dashboard
} else {
  const error = await response.json();
  // Show error message
}
```

### 4. Authenticated Requests
```javascript
const token = localStorage.getItem('jwt_token');

const response = await fetch('/api/v1/clusters', {
  method: 'GET',
  headers: {
    'Authorization': `Bearer ${token}`
  }
});

if (response.status === 401) {
  // Token expired or invalid, redirect to login
  localStorage.clear();
  window.location.href = '/login';
} else if (response.status === 403) {
  // Insufficient permissions
  alert('You do not have permission to perform this action');
} else if (response.ok) {
  const data = await response.json();
  // Process data
}
```

### 5. Role-Based UI
```javascript
const userRole = localStorage.getItem('user_role');

// Show/hide admin-only features
if (userRole === 'admin') {
  // Show: Create/Edit/Delete buttons
  document.querySelector('.admin-panel').style.display = 'block';
} else {
  // Hide admin features, show read-only view
  document.querySelector('.admin-panel').style.display = 'none';
}
```

### 6. Logout
```javascript
function logout() {
  localStorage.clear();
  window.location.href = '/login';
}
```

---

## Token Expiration

**Token Duration**: 24 hours

**Handling Expiration**:
- Token expiration time is included in login response (`expiresAt`)
- When token expires, API returns `401 Unauthorized`
- Frontend should:
  1. Detect 401 responses
  2. Clear stored token
  3. Redirect to login page
  4. Optionally: show "Session expired" message

**Example Expiration Handler**:
```javascript
async function fetchWithAuth(url, options = {}) {
  const token = localStorage.getItem('jwt_token');

  const response = await fetch(url, {
    ...options,
    headers: {
      ...options.headers,
      'Authorization': `Bearer ${token}`
    }
  });

  if (response.status === 401) {
    // Token expired or invalid
    localStorage.clear();
    window.location.href = '/login?expired=true';
    throw new Error('Session expired');
  }

  return response;
}
```

---

## Error Handling

All API errors follow this format:

```json
{
  "error": "error_code",
  "message": "Human-readable error message"
}
```

**Common Error Codes**:
- `validation_error` - Invalid input data
- `unauthorized` - Missing or invalid authentication
- `forbidden` - Insufficient permissions
- `invalid_credentials` - Wrong email/password
- `user_exists` - User already registered
- `account_disabled` - User account is disabled
- `internal_error` - Server error
- `method_not_allowed` - Wrong HTTP method
- `not_found` - Resource not found

---

## Security Considerations

1. **HTTPS**: Always use HTTPS in production to protect tokens in transit
2. **Token Storage**: Use `sessionStorage` instead of `localStorage` for better security
3. **Token Refresh**: Tokens expire after 24 hours, users must re-login
4. **Password Requirements**: Minimum 8 characters (can be enhanced with complexity rules)
5. **CORS**: Configure CORS properly if frontend is on different domain
6. **XSS Protection**: Sanitize all user inputs to prevent XSS attacks

---

## Testing Credentials

For development/testing, you can create a test admin:

```bash
# Register first admin (no auth required)
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "userId": "[email protected]",
    "password": "TestPassword123",
    "name": "Test",
    "surname": "Admin",
    "role": "admin"
  }'

# Login
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "userId": "[email protected]",
    "password": "TestPassword123"
  }'
```
