# Frontend Authentication Implementation Tasks

## Overview

This document outlines the tasks required to implement the authentication system in the krkn-operator-console frontend.

**Total Tasks**: 9
- **Priority 1 (P1)**: 7 tasks
- **Priority 2 (P2)**: 2 tasks

**API Documentation**: See `docs/api-authentication-spec.md` for complete API reference

---

## Task Dependency Graph

```
┌─────────────────────────────────────────────┐
│ 1. Setup Authentication Infrastructure     │
│    (krkn-operator-b3n) - P1                │
└─────────────────┬───────────────────────────┘
                  │
        ┌─────────┴────────┬──────────────────┐
        │                  │                  │
        ▼                  ▼                  ▼
┌──────────────┐  ┌─────────────┐  ┌──────────────┐
│ 2. Admin     │  │ 3. First    │  │ 4. Login     │
│ Check Page   │  │ Admin Reg   │  │ Page         │
│ (j01) - P1   │  │ (4ju) - P1  │  │ (up8) - P1   │
└──────────────┘  └─────────────┘  └──────┬───────┘
                                           │
                           ┌───────────────┼───────────────┐
                           │               │               │
                           ▼               ▼               ▼
                    ┌─────────────┐ ┌─────────────┐ ┌─────────────┐
                    │ 5. Protected│ │ 6. Session  │ │ 7. Error    │
                    │ Routes      │ │ Expiration  │ │ Handling    │
                    │ (jkx) - P1  │ │ (urc) - P1  │ │ (2q3) - P2  │
                    └──────┬──────┘ └──────┬──────┘ └─────────────┘
                           │               │
                           ▼               │
                    ┌─────────────┐        │
                    │ 8. Role UI  │        │
                    │ (6q1) - P1  │        │
                    └──────┬──────┘        │
                           │               │
                           └───────┬───────┘
                                   │
                                   ▼
                            ┌─────────────┐
                            │ 9. E2E Tests│
                            │ (82z) - P2  │
                            └─────────────┘
```

---

## Implementation Order

### Phase 1: Foundation (Week 1)
**Start with these tasks - they are the foundation:**

1. **Setup Authentication Infrastructure** (krkn-operator-b3n) - P1
   - **Blocks**: Tasks 2, 3, 4
   - **Estimated effort**: 2-3 days
   - **Key deliverables**:
     - Auth context/state management
     - Token storage utilities
     - API client with JWT injection
     - 401 auto-handling

### Phase 2: Core Authentication UI (Week 1-2)
**Work on these in parallel after Phase 1:**

2. **Admin Registration Check Page** (krkn-operator-j01) - P1
   - **Depends on**: Task 1
   - **Estimated effort**: 1 day
   - **Key deliverables**: Initial routing logic

3. **First Admin Registration Page** (krkn-operator-4ju) - P1
   - **Depends on**: Task 1
   - **Estimated effort**: 2 days
   - **Key deliverables**: Registration form with validation

4. **Login Page** (krkn-operator-up8) - P1
   - **Depends on**: Task 1
   - **Blocks**: Tasks 5, 6, 7
   - **Estimated effort**: 2 days
   - **Key deliverables**: Login form, token handling

### Phase 3: Advanced Features (Week 2-3)
**Work on these after Phase 2:**

5. **Protected Routes and Navigation Guards** (krkn-operator-jkx) - P1
   - **Depends on**: Task 4
   - **Blocks**: Task 8
   - **Estimated effort**: 2 days
   - **Key deliverables**: Route protection, auth guards

6. **Session Expiration and Logout** (krkn-operator-urc) - P1
   - **Depends on**: Task 4
   - **Blocks**: Task 9
   - **Estimated effort**: 2 days
   - **Key deliverables**: Session timer, logout functionality

7. **Authentication Error Handling** (krkn-operator-2q3) - P2
   - **Depends on**: Task 4
   - **Estimated effort**: 2 days
   - **Key deliverables**: Error display, retry logic

### Phase 4: Role-Based Features (Week 3)
**Complete the authentication system:**

8. **Role-Based UI Components** (krkn-operator-6q1) - P1
   - **Depends on**: Task 5
   - **Blocks**: Task 9
   - **Estimated effort**: 2 days
   - **Key deliverables**: AdminOnly component, permission hooks

### Phase 5: Testing (Week 3-4)
**Final phase - comprehensive testing:**

9. **Authentication E2E Tests** (krkn-operator-82z) - P2
   - **Depends on**: Tasks 8, 6
   - **Estimated effort**: 3 days
   - **Key deliverables**: Complete E2E test suite

---

## Quick Start for Frontend Developers

### 1. Read the API Documentation
Start by reading `docs/api-authentication-spec.md` to understand:
- Available endpoints
- Request/response formats
- Error codes
- JWT token usage

### 2. Check Available Work
```bash
bd ready
```

### 3. Start with Foundation Task
The first task to work on is **krkn-operator-b3n** (Setup Authentication Infrastructure).
This task blocks all other authentication work.

### 4. Claim a Task
```bash
bd update krkn-operator-b3n --status=in_progress
```

### 5. View Task Details
```bash
bd show krkn-operator-b3n
```

---

## Task Details Quick Reference

| Task ID | Title | Priority | Depends On | Estimated |
|---------|-------|----------|------------|-----------|
| krkn-operator-b3n | Auth Infrastructure | P1 | None | 2-3 days |
| krkn-operator-j01 | Admin Check Page | P1 | b3n | 1 day |
| krkn-operator-4ju | First Admin Registration | P1 | b3n | 2 days |
| krkn-operator-up8 | Login Page | P1 | b3n | 2 days |
| krkn-operator-jkx | Protected Routes | P1 | up8 | 2 days |
| krkn-operator-urc | Session Expiration | P1 | up8 | 2 days |
| krkn-operator-2q3 | Error Handling | P2 | up8 | 2 days |
| krkn-operator-6q1 | Role-Based UI | P1 | jkx | 2 days |
| krkn-operator-82z | E2E Tests | P2 | 6q1, urc | 3 days |

**Total Estimated Effort**: 18-20 days (for 1 developer) or 2-3 weeks (for 2 developers in parallel)

---

## API Endpoints Summary

### Public Endpoints (No Auth Required)
- `GET /api/v1/auth/is-registered` - Check if admin exists
- `POST /api/v1/auth/register` - Register new user
- `POST /api/v1/auth/login` - Login and get JWT token

### Protected Endpoints (Require JWT)
All other endpoints require JWT token in `Authorization: Bearer <token>` header.

**User + Admin Access**:
- All GET operations
- Scenario operations

**Admin Only**:
- POST/PUT/DELETE on `/operator/targets`
- POST on `/provider-config`
- PATCH on `/providers`

---

## Testing API Locally

### 1. Start the Operator
```bash
make run
```

### 2. Test Registration (First Admin)
```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "userId": "[email protected]",
    "password": "TestPassword123",
    "name": "Test",
    "surname": "Admin",
    "role": "admin"
  }'
```

### 3. Test Login
```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "userId": "[email protected]",
    "password": "TestPassword123"
  }'
```

Save the token from the response.

### 4. Test Protected Endpoint
```bash
export TOKEN="<token-from-login>"
curl -X GET http://localhost:8080/api/v1/clusters \
  -H "Authorization: Bearer $TOKEN"
```

---

## Common Issues and Solutions

### CORS Errors
If running frontend on different port, configure CORS on the backend.

### Token Not Working
- Check token format: must be `Bearer <token>` (with space)
- Check token expiration (24 hours)
- Verify token is stored correctly in sessionStorage

### 401 on Protected Routes
- Ensure token is included in request headers
- Check if token has expired
- Verify API endpoint is correct

### Role Permissions Not Working
- Check user role in JWT claims
- Verify role-based checks in UI components
- Test with both admin and user accounts

---

## Questions?

For technical questions about the API:
- See `docs/api-authentication-spec.md`

For beads/task questions:
- Run `bd prime` for workflow details
- Check `AGENTS.md` for beads quick reference

For architecture questions:
- See `CLAUDE.md` for project guidelines
- Check `pkg/auth/doc.go` for authentication utilities documentation
