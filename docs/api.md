# KlipForge API

Base URL: `/api/v1`. JSON errors use:

```json
{"error":{"code":"AUTHENTICATION_FAILED","message":"Authentication failed.","request_id":"request-id"}}
```

All responses include `X-Request-ID`. Authentication request bodies reject unknown JSON fields and trailing JSON values.

## Register

`POST /api/v1/auth/register`

```json
{"email":"creator@example.com","password":"StrongPass1","full_name":"Ari Lin","role":"CLIPPER"}
```

`role` is limited to `BRAND` or `CLIPPER`. Success is `201 Created`, sets the refresh cookie, and returns the access token, safe user data, and session expiry:

```json
{"access_token":"...","token_type":"Bearer","expires_in":900,"user":{"id":"uuid","email":"creator@example.com","role":"CLIPPER","is_active":true,"display_name":"Ari Lin","avatar_url":null,"bio":null,"created_at":"..."},"session":{"id":"uuid","expires_at":"..."}}
```

Relevant errors: `VALIDATION_ERROR`, `EMAIL_ALREADY_REGISTERED`, `RATE_LIMIT_EXCEEDED`.

## Login

`POST /api/v1/auth/login`

```json
{"email":"clipper@klipforge.local","password":"Clipper123!"}
```

Success is `200 OK` with the same token response and refresh cookie as registration. Unknown accounts and wrong passwords both return `AUTHENTICATION_FAILED`. Inactive accounts return `USER_INACTIVE`.

The seeded credentials are local development fixtures only: `admin@klipforge.local` / `Admin123!`, `brand@klipforge.local` / `Brand123!`, and `clipper@klipforge.local` / `Clipper123!`.

## Refresh

`POST /api/v1/auth/refresh`

No JSON body is required. The endpoint reads the HttpOnly refresh cookie, rotates it transactionally, replaces the cookie, and returns a new access token. Relevant errors include `REFRESH_TOKEN_MISSING`, `REFRESH_TOKEN_INVALID`, `REFRESH_TOKEN_EXPIRED`, `REFRESH_TOKEN_REUSED`, and `USER_INACTIVE`.

An old token cannot be used after rotation. Reuse revokes its complete session and clears the cookie.

## Logout

`POST /api/v1/auth/logout`

Revokes the session identified by the refresh cookie, clears that cookie, and returns `204 No Content`. Missing or previously cleared cookies remain a successful no-op.

## Logout all

`POST /api/v1/auth/logout-all` with `Authorization: Bearer <access-token>`.

Revokes all refresh sessions for the authenticated user, clears the current cookie, and returns `204 No Content`.

## Current user

`GET /api/v1/auth/me` with `Authorization: Bearer <access-token>`.

Success is `200 OK`:

```json
{"user":{"id":"uuid","email":"clipper@klipforge.local","role":"CLIPPER","is_active":true,"display_name":"Ari Lin","avatar_url":null,"bio":"...","created_at":"..."}}
```

Protected endpoints distinguish `ACCESS_TOKEN_MISSING`, `ACCESS_TOKEN_INVALID`, `ACCESS_TOKEN_EXPIRED`, `SESSION_REVOKED`, and `USER_INACTIVE`.

## Token and cookie configuration

Access tokens default to 15 minutes (`ACCESS_TOKEN_TTL`); refresh sessions default to 30 days (`REFRESH_TOKEN_TTL`). See [security.md](./security.md) and `.env.example` for issuer, audience, secret, cookie, bcrypt, and throttling configuration.

