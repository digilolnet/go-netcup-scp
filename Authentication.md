# Authentication

API Server: https://www.servercontrolpanel.de/scp-core

To use the API, you need a bearer access token. Each request must include the bearer access token in the HTTP Authorization header. There are three relevant processes for handling access tokens:

- Generate offline access refresh token
- Refresh access token
- Revoke refresh token

**Generate offline access refresh token**

- Generate verification URI.

```bash
curl -X POST 'https://www.servercontrolpanel.de/realms/scp/protocol/openid-connect/auth/device' \
  -d "client_id=scp" \
  -d 'scope=offline_access openid' | jq
```

```json
{
  "device_code": "<device-code>",
  "user_code": "<user-code>",
  "verification_uri": "https://www.servercontrolpanel.de/realms/scp/device",
  "verification_uri_complete": "https://www.servercontrolpanel.de/realms/scp/device?user_code=<user-code>",
  "expires_in": 600,
  "interval": 5
}
```

- Open verification_uri_complete from response in Browser and enter your user credentials.
- Accept grant access to SCP with privileges profile, offline_access, roles, email after submitting your credentials.
- Generate access token and offline access refresh token with device_code from response in step "Generate verification URI".

```bash
curl -X POST 'https://www.servercontrolpanel.de/realms/scp/protocol/openid-connect/token' \
  -d 'grant_type=urn:ietf:params:oauth:grant-type:device_code' \
  -d 'device_code=<device-code>' \
  -d 'client_id=scp' | jq
```

```json
{
  "access_token": "<access-token>",
  "expires_in": 300,
  "refresh_expires_in": 0,
  "refresh_token": "<refresh-token>",
  "token_type": "Bearer",
  "not-before-policy": 0,
  "session_state": "<session-state>",
  "scope": "profile offline_access email"
}
```

- Use access token within the next 300 seconds to access the API. See "Refresh access token" how to obtain a new access token.

```bash
curl 'https://www.servercontrolpanel.de/scp-core/api/v1/servers?limit=10' \
  -H 'Authorization: Bearer <access-token>' | jq
...
```

**Refresh access token**

- The offline refresh token can be used multiple times and does not expire as long as it is used at least once every 30 days.

```bash
curl 'https://www.servercontrolpanel.de/realms/scp/protocol/openid-connect/token' \
  -d 'client_id=scp'\
  -d 'refresh_token=<refresh_token>' \
  -d 'grant_type=refresh_token' | jq
```

**Revoke refresh token**

- If the refresh token is leaked or no longer needed it could be revoked.

```bash
curl 'https://www.servercontrolpanel.de/realms/scp/protocol/openid-connect/revoke' \
  -d 'client_id=scp' \
  -d "token=<refresh-token>" \
  -d "token_type_hint=refresh_token" | jq
...
```

- Forgotten refresh tokens can be revoked in the Account Console (Menu "Applications" - Application "scp" - Button "Remove access"): Open

**User Info**

The numeric user ID (used in API paths like `/users/{userId}/...`) is embedded
in the JWT access token as the `id` claim. The Go library extracts it directly
from the token without an extra HTTP request. Note that the standard `sub`
claim is the Keycloak account UUID, not the numeric SCP user ID.

For reference, the userinfo endpoint returns the same claims:

```bash
curl 'https://www.servercontrolpanel.de/realms/scp/protocol/openid-connect/userinfo' \
  -H 'Authorization: Bearer <access-token>' | jq
{
  ...
  "sub": "<user-id>"
  ...
}
```

**Auto-refresh**

The Go `auth.Manager` keeps the access token fresh two ways:

- On demand: every API request obtains its token via `ValidAccessToken`, which
  returns the cached token or refreshes it first. Concurrent callers share a
  single refresh (single-flight), so parallel requests never race the IdP.
- Proactively: a timer refreshes 30 seconds before expiry (floored at 15s, and
  re-armed after transient failures) — this matters for long-lived processes
  such as the VNC console bridge.

An optional callback persists refreshed tokens. When persisting, the library
adds an `obtained_at` timestamp (not part of the IdP response) so a token
reloaded from disk expires at the correct absolute time; token files without
it are treated as expired and refreshed on first use. `Close` stops all
refreshing; no token can be fetched or persisted afterwards.
