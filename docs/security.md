# KlipForge security model

## Authentication boundary

Phase 2B establishes identity. Phase 2C separately enforces roles, current account/session state, and PostgreSQL-backed resource ownership. Role checks are reusable guards rather than handler-local conditionals.

## Passwords and login

- Registration accepts `BRAND` and `CLIPPER`; public `ADMIN` creation is rejected.
- Email addresses are trimmed and lowercased before storage or lookup.
- Passwords require 8–72 bytes, one uppercase letter, one lowercase letter, and one number. The upper bound prevents bcrypt input truncation.
- Passwords use bcrypt with `BCRYPT_COST` (default `12`). Plaintext passwords and password hashes are never logged or returned.
- Unknown-email and wrong-password login attempts return the same `AUTHENTICATION_FAILED` response. A fixed bcrypt comparison also reduces account-enumeration timing differences.

## Access tokens

Access tokens are short-lived HS256 JWTs returned in JSON. The default lifetime is 15 minutes. The API requires and validates `sub`, `role`, `session_id`, `jti`, `iss`, `aud`, `iat`, and `exp`; it rejects malformed UUID claims, incorrect issuer or audience, expired tokens, invalid signatures, and every signing algorithm other than HS256.

Protected endpoints receive the token as `Authorization: Bearer <access-token>`. Middleware also checks that the user remains active and the referenced database session still has an active refresh token. The future frontend must keep access tokens in memory, not local storage.

The request context stores only a typed safe principal: user ID, role, session ID, JWT ID, and current account/session flags. Raw claims and tokens are not placed in context logs.

## Authorization and ownership

The only accepted roles are `ADMIN`, `BRAND`, and `CLIPPER`. Role values are validated at registration, JWT validation, authentication-context creation, middleware enforcement, and policy entry points.

Ownership decisions query trusted PostgreSQL relationships with parameterized statements:

- campaign `brand_id` for campaign management and participant/submission/payout visibility;
- campaign participation `clipper_id` for participation and submission ownership;
- payout participation and campaign relationships for clipper and brand visibility;
- profile `user_id` for private profile access.

Client-supplied owner IDs are never authorization evidence. Swapping a route resource ID therefore causes a new database ownership lookup rather than inheriting access from another request.

Ownership failures normally return the same 404 response as a missing resource. This prevents callers from distinguishing another tenant's private resource from a nonexistent identifier. Pure role denials return 403. Missing/invalid identity and revoked sessions return 401; inactive accounts retain the existing 403 `USER_INACTIVE` response.

The current identity schema models availability with `users.is_active`; it has no separate suspended status. A future suspension state must be incorporated into the same current-database account check before protected access.

`ADMIN` exceptions are declared by individual policies. Administrators remain subject to authentication, current account state, and active-session checks. Policy evaluation alone does not create an audit entry; the future service performing a sensitive administrative mutation must write the durable audit record in the same business transaction. This avoids audit noise from ordinary reads and denied probes.

## Refresh sessions and rotation

Refresh tokens are opaque 256-bit random values. Only an HMAC-SHA256 digest, keyed by `REFRESH_TOKEN_PEPPER`, is stored in PostgreSQL. Each row records a session ID, token family, expiry, rotation/revocation state, replacement link, user agent, and IP address.

Rotation is a PostgreSQL transaction:

1. Hash the cookie token and lock its database row with `SELECT ... FOR UPDATE`.
2. Confirm the token, account, expiry, and session are valid.
3. Insert one replacement in the same session and family.
4. Mark the old row rotated/revoked and link it to the replacement.
5. Commit the replacement and audit record atomically.

The row lock prevents concurrent requests from creating two active replacements. Presenting an already rotated or revoked token is treated as possible theft: every token in the session is revoked, an `auth.refresh_reuse` audit event is committed, the cookie is cleared, and `REFRESH_TOKEN_REUSED` is returned.

Logout is idempotent and revokes the current database session. Authenticated logout-all revokes every refresh session for the user. Revoked sessions cannot refresh and their access tokens fail the session-state check.

## Refresh cookie

The default cookie is `klipforge_refresh_token`, with `HttpOnly`, `SameSite=Lax`, and path `/api/v1/auth`. Its Max-Age follows `REFRESH_TOKEN_TTL` (default 30 days). `Secure` defaults to true in production and is false in the checked-in local development example. An optional domain must be configured explicitly. Cookie deletion uses the same name, path, domain, SameSite, and Secure settings.

## Rate limiting and logging

Login, registration, and refresh use IP-scoped Redis counters. Defaults are 10, 5, and 30 requests per one-minute window. If Redis is unavailable, authentication endpoints fail closed with `AUTH_SERVICE_UNAVAILABLE`; they do not bypass throttling.

Request logging records method, route, status, duration, request ID, and remote IP. It never records request bodies, Authorization headers, Cookie headers, passwords, raw tokens, or token/password hashes. Audit metadata is deliberately limited to safe identifiers and outcomes.

Authorization denials log only request ID, route, method, authenticated user ID, role, policy name, decision, and a safe reason code. Database error details are not exposed to clients. Authentication rate limits remain unchanged and execute on their existing public endpoints before authentication services.

## Secrets

`JWT_SECRET` and `REFRESH_TOKEN_PEPPER` must each contain at least 32 characters and must be independently generated. Development fallback values are rejected when `APP_ENV=production`. No production secrets belong in committed files.
