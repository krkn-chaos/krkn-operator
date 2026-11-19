# Optional Keycloak OIDC authentication (operator POC)

This implementation adds Keycloak login to the **krkn-operator backend**. It does not configure Keycloak, LDAP federation, or the web console. Local password login and existing API Bearer JWTs remain available.

## Configuration

The operator reads these optional environment variables. Supply client secrets through Kubernetes Secret references, for example using Helm `operator.extraEnv`. If the login variables are absent, Keycloak login responds with `503`; if the Admin API variables are absent, group discovery is unavailable.

| Variable | Purpose |
| --- | --- |
| `KEYCLOAK_ISSUER_URL` | Exact issuer, e.g. `https://id.example/realms/production` |
| `KEYCLOAK_OIDC_CLIENT_ID` | OIDC client for human login |
| `KEYCLOAK_OIDC_CLIENT_SECRET` | Secret for the confidential login client (required in this POC) |
| `KEYCLOAK_OIDC_REDIRECT_URL` | Exact callback URL ending in `/api/v1/auth/oidc/keycloak/callback` |
| `KEYCLOAK_ADMIN_CLIENT_ID` | Separate service client for group discovery |
| `KEYCLOAK_ADMIN_CLIENT_SECRET` | Service client secret with group read access |
| `WEBSOCKET_ALLOWED_ORIGINS` | Additional trusted console origins if different from the API host |

Example Helm values fragment (the Secret must already exist):

```yaml
operator:
  extraEnv:
    - name: KEYCLOAK_ISSUER_URL
      value: https://id.example/realms/production
    - name: KEYCLOAK_OIDC_CLIENT_ID
      value: krkn-operator-login
    - name: KEYCLOAK_OIDC_CLIENT_SECRET
      valueFrom:
        secretKeyRef:
          name: krkn-keycloak-clients
          key: oidc-client-secret
    - name: KEYCLOAK_OIDC_REDIRECT_URL
      value: https://operator.example/api/v1/auth/oidc/keycloak/callback
    - name: KEYCLOAK_ADMIN_CLIENT_ID
      value: krkn-operator-groups
    - name: KEYCLOAK_ADMIN_CLIENT_SECRET
      valueFrom:
        secretKeyRef:
          name: krkn-keycloak-clients
          key: admin-client-secret
```

The Keycloak login client must emit the user's groups in the ID token's `groups` claim as **full paths**, such as `/Engineering/Ops`. The Admin API must expose the same paths. The separate service client needs read access to realm groups. The operator does not need access to any LDAP server behind Keycloak.

## Login and sessions

- `GET /api/v1/auth/oidc/keycloak/login` starts Authorization Code + PKCE, state, and nonce.
- `GET /api/v1/auth/oidc/keycloak/callback` exchanges the code, verifies the ID token, creates a local `KrknUser` record on first login, and issues an operator session cookie. The browser does not receive Keycloak tokens.
- `POST /api/v1/auth/login` still returns the existing Bearer JWT and now also sets a browser session cookie.
- `GET /api/v1/auth/session` returns the current principal and a CSRF token. Browser requests using the cookie must send `X-CSRF-Token` with that value for methods other than GET, HEAD, and OPTIONS. Bearer requests are unchanged.
- `POST /api/v1/auth/logout` clears the browser cookie; this is not a server-side revocation of a copied token.

The cookie is `HttpOnly`, `Secure`, `SameSite=Lax`, signed with the existing operator JWT key, and valid for eight hours. HTTPS is required for browser use. There is no new session database. External sessions include the account's Kubernetes UID so deleting and recreating the account invalidates old sessions.

An OIDC `KrknUser` has `spec.authProvider`, `spec.externalIssuer`, `spec.externalSubject`, and display-only username/email snapshots. It has no password Secret and receives only the `user` role. Its `userId` is a deterministic hash of provider, issuer, and subject; matching by email is forbidden.

## Groups and permissions

`KrknUserGroup.spec.externalGroups` accepts multiple `{provider, group}` entries. Each entry is an exact groups-claim value. A matching external group grants that local group's cluster permissions. Unmapped groups grant nothing. Local users continue to obtain group memberships from their `KrknUser` labels. A mapping edit is evaluated on the next request; a Keycloak membership change requires a fresh login because group claims are in the session.

Admin-only discovery endpoint:

```text
GET /api/v1/identity-providers/keycloak/groups?search=ops&first=0&max=50
```

It returns `groups[]` with provider, Keycloak ID, path, and `claimValue` (the path to save in a mapping). It uses a service-account token and propagates provider failures as `502`. `pkg/identity.GroupDirectory` is the public provider adapter contract; `pkg/identity.OIDCAuthenticator` is the public login contract. A Dex implementation can use the generic OIDC client for login and provide its own group directory adapter.

## Per-user controls and blacklist

An admin can set `active=false` through the existing user PATCH endpoint. The operator checks the active state on every external HTTP request; an inactive account cannot use an existing cookie.

- `DELETE /api/v1/users/{userId}` removes the account. A later valid OIDC login may create a new record. The old session remains invalid because the Kubernetes UID changes.
- `DELETE /api/v1/users/{userId}?blacklist=true` first writes a per-identity Kubernetes Secret, then deletes the account. If writing the block fails, deletion does not start. A block also prevents first-login provisioning and invalidates existing sessions.
- `GET /api/v1/identity-blocks` lists blocked identities for admins, including the saved username/email snapshot.
- `DELETE /api/v1/identity-blocks/{id}` removes a block. Only an admin can do this.

The block key is the exact `{provider, issuer, sub}` tuple. Username and email are display metadata and never participate in authorization. A Keycloak account recreated with a different `sub` is a different identity and is not covered by the old block.

Cookie WebSockets require a trusted `Origin` and use the same account/block check at handshake. Active v2 sockets recheck before messages and pings; log sockets recheck every five seconds and close when blocked. Existing Bearer-subprotocol WebSocket clients retain their previous behavior.

## Current POC limits

The console has not yet been updated with the Keycloak button, group explorer, blacklist view, or the unmapped-groups empty state. The backend endpoints are ready for that integration. A live Keycloak deployment has not been exercised; the tests use a simulated OIDC issuer and Admin API.
