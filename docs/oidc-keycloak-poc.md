# Optional Keycloak OIDC authentication (operator POC)

This implementation adds Keycloak login to the **krkn-operator backend**. It does not configure the Keycloak server, LDAP federation, or the web console. Local password login and existing API Bearer JWTs remain available.

## Identity provider management

An operator admin can create and manage multiple Keycloak instances without a new persistence service. Each configuration is an opaque Kubernetes Secret in the operator namespace. The immutable `id` identifies the instance in login URLs, user records, blocks, and group mappings; `type` is currently `keycloak`. The issuer is immutable after creation. To move to another realm, choose a new ID and create new mappings. Provider deletion is intentionally unavailable so an ID cannot be reused while users or grants still refer to it.

Admin API (requires an operator admin session or Bearer token):

```text
GET   /api/v1/identity-providers
POST  /api/v1/identity-providers
GET   /api/v1/identity-providers/{id}
PATCH /api/v1/identity-providers/{id}
GET   /api/v1/identity-providers/{id}/groups?search=ops&first=0&max=50
```

Create body example:

```json
{
  "id": "corp-keycloak",
  "type": "keycloak",
  "displayName": "Corporate Keycloak",
  "issuer": "https://id.example/realms/production",
  "clientId": "krkn-operator-login",
  "clientSecret": "<login-client-secret>",
  "redirectUrl": "https://operator.example/api/v1/auth/oidc/corp-keycloak/callback",
  "adminClientId": "krkn-operator-groups",
  "adminClientSecret": "<group-reader-secret>",
  "enabled": true
}
```

`PATCH` accepts `displayName`, client IDs/secrets, `redirectUrl`, and `enabled`; omitted secrets retain their existing values. The admin responses contain credential-presence flags but never the secret values. Group discovery credentials are optional until group import is needed. A disabled provider disappears from the public login list, rejects new login/callback requests, and causes existing HTTP and WebSocket sessions from that provider to fail the live identity check. Every enable/disable transition rotates a private session epoch, so old sessions stay invalid after re-enabling; users must log in again.

The public login-choice endpoint is `GET /api/v1/auth/providers`; it returns only enabled providers with `id`, `type`, `displayName`, and `loginUrl`. The console should render its "Login with …" buttons from this response.

## Configuration

The previous environment-variable setup remains available as a read-only provider with ID `keycloak` when no Secret-backed provider with that ID exists. Supply client secrets through Kubernetes Secret references, for example using Helm `operator.extraEnv`. A Secret-backed `keycloak` configuration with the same issuer overrides it. New installations should use the admin API above.

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

## Containerized integration test

With Podman running, execute this from the operator repository root:

```sh
go test -tags=integration ./internal/api -run '^TestKeycloakOIDCIntegration$' -count=1 -v
```

The test uses a disposable `quay.io/keycloak/keycloak:26.8.2` container on an automatically assigned loopback port. It creates its own realm, confidential login client, read-only group client, groups mapper, group and user. It then completes the real Authorization Code + PKCE flow against the operator OIDC handlers, checks the operator session and external group mapping, and queries the Keycloak Admin API for the group. The container is removed when the test ends. The normal `go test ./internal/api` suite does not start Keycloak. The PR workflow runs this integration test with Docker; locally it defaults to Podman. Set `KRKN_KEYCLOAK_TEST_RUNTIME=docker` to use Docker locally or `KRKN_KEYCLOAK_TEST_IMAGE` to use another compatible image.

## Login and sessions

- `GET /api/v1/auth/oidc/{id}/login` starts Authorization Code + PKCE, state, and nonce for the configured instance. The login state is bound to that provider ID.
- `GET /api/v1/auth/oidc/{id}/callback` exchanges the code, verifies the ID token, creates a local `KrknUser` record on first login, records `lastLogin`, and issues an operator session cookie. The browser does not receive Keycloak tokens.
- `POST /api/v1/auth/login` still returns the existing Bearer JWT and now also sets a browser session cookie.
- `GET /api/v1/auth/session` returns the current principal, local `mappedGroups` IDs, and a CSRF token. An external user with an empty `mappedGroups` array has no mapped group grants; the console can show its empty state. Browser requests using the cookie must send `X-CSRF-Token` with that value for methods other than GET, HEAD, and OPTIONS. Bearer requests are unchanged.
- `POST /api/v1/auth/logout` clears the browser cookie; this is not a server-side revocation of a copied token.

The cookie is `HttpOnly`, `Secure`, `SameSite=Lax`, signed with the existing operator JWT key, and valid for eight hours. HTTPS is required for browser use. There is no new session database. External sessions include the account's Kubernetes UID so deleting and recreating the account invalidates old sessions.

An OIDC `KrknUser` has `spec.authProvider`, `spec.externalIssuer`, `spec.externalSubject`, and display-only username/email snapshots. It has no password Secret and receives only the `user` role. Its `userId` is a deterministic hash of provider, issuer, and subject; matching by email is forbidden.

## Groups and permissions

`KrknUserGroup.spec.externalGroups` accepts multiple `{provider, group, groupId}` entries. `provider` is the immutable configuration ID, not the product type; `group` is an exact groups-claim value (a full Keycloak path); `groupId` is optional remote metadata for display and rename diagnostics. Authorization compares provider ID and claim value together. For example, `/Ops` from `corp-keycloak` cannot satisfy a mapping for `partner-keycloak`. A matching external group grants that local group's cluster permissions. Unmapped groups grant nothing. Local users continue to obtain group memberships from their `KrknUser` labels. A mapping edit is evaluated on the next request; a Keycloak membership change requires a fresh login because group claims are in the session.

The group API's `memberCount` still counts local KrknUser label assignments only; remote membership is not persisted in the operator.

Admin-only discovery endpoint:

```text
GET /api/v1/identity-providers/corp-keycloak/groups?search=ops&first=0&max=50
```

It returns `groups[]` with provider ID, Keycloak group ID, path, and `claimValue` (the path to save as `group`). It uses separate service-account credentials and propagates provider failures as `502`. `pkg/identity.GroupDirectory` is the public provider adapter contract; `pkg/identity.OIDCAuthenticator` is the public login contract. A Dex implementation can use the generic OIDC client for login and provide its own group directory adapter.

## Per-user controls and blacklist

An admin can set `active=false` through the existing user PATCH endpoint. The operator checks the active state on every external HTTP request; an inactive account cannot use an existing cookie.

`GET /api/v1/users?authProvider={id}` lists current accounts for one provider, including username/email snapshots, active state and last login. Deleted accounts are absent; no historical login database is maintained.

- `DELETE /api/v1/users/{userId}` removes the account. A later valid OIDC login may create a new record. The old session remains invalid because the Kubernetes UID changes.
- `DELETE /api/v1/users/{userId}?blacklist=true` first writes a per-identity Kubernetes Secret, then deletes the account. If writing the block fails, deletion does not start. A block also prevents first-login provisioning and invalidates existing sessions.
- `GET /api/v1/identity-blocks?authProvider={id}` lists blocked identities for admins, optionally filtered by provider ID, including the saved username/email snapshot.
- `DELETE /api/v1/identity-blocks/{id}` removes a block. Only an admin can do this.

The block key is the exact `{provider, issuer, sub}` tuple. Username and email are display metadata and never participate in authorization. A Keycloak account recreated with a different `sub` is a different identity and is not covered by the old block.

Cookie WebSockets require a trusted `Origin` and use the same account/block check at handshake. Active v2 sockets recheck before messages and pings; log sockets recheck every five seconds and close when blocked. Existing Bearer-subprotocol WebSocket clients retain their previous behavior.

## Current POC limits

The console has not yet been updated with the Keycloak button, group explorer, blacklist view, or the unmapped-groups empty state. The backend endpoints are ready for that integration. A live Keycloak deployment has not been exercised; the tests use a simulated OIDC issuer and Admin API.
