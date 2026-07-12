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

## Authorization

Authentication establishes a typed principal containing the current user ID, validated role, session ID, JWT ID, account state, and session state. Authorization is a separate guard and policy layer. No Phase 2C test routes or incomplete product routes are exposed by the production router.

### Status behavior

- `401 Unauthorized`: authentication is missing or invalid, an access token expired, or its session was revoked.
- `403 Forbidden`: identity is valid, but the role or business rule does not permit the operation. Codes include `FORBIDDEN` and `ROLE_NOT_ALLOWED`.
- `404 Not Found`: the resource is absent or is intentionally hidden because the caller does not own it. Ownership policies normally use `RESOURCE_NOT_FOUND` to reduce identifier enumeration.

Every error retains the standard request-ID envelope. Database errors and ownership identifiers are never returned.

### Authorization matrix

| Capability | ADMIN | BRAND | CLIPPER |
| --- | --- | --- | --- |
| Manage campaigns | Any campaign, explicit admin policy | Own campaigns only | Never |
| View campaigns | Any campaign | Own campaigns | Public active campaigns |
| Join campaigns | No automatic participation | Never | Active campaigns when rules permit |
| View participants | Inspection endpoints | Own campaigns | Own participation only |
| Manage submissions | Inspection endpoints; no moderation in Phase 3B | View own campaign submissions | Own PENDING submission only |
| Review submissions | Any through moderation policy | Own campaigns only | Never |
| View payouts | Any | Own campaign obligations | Own payouts only |
| Process payouts | Only role permitted | Never | Never |
| Private profiles | Administrative view | Self only | Self only |
| Update profiles | Self only | Self only | Self only |

Administrator access is explicit per policy; there is no blanket bypass.

### Future route integration

Product routes should compose the reusable guards and a database-backed policy without duplicating role checks:

```go
router.With(
    authorization.RequireAuthentication,
    authorization.RequireActiveUser,
    authorization.RequireActiveSession,
    authorization.RequireAnyRole(auth.RoleAdmin, auth.RoleBrand),
    authorization.RequireOwnership("campaign.manage", campaignOwnershipCheck),
).Patch("/campaigns/{campaignID}", handler)
```

The ownership check must pass the route resource ID to `Policies.CanManageCampaign`; it must never accept a client-supplied owner ID as proof of ownership.

## Token and cookie configuration

Access tokens default to 15 minutes (`ACCESS_TOKEN_TTL`); refresh sessions default to 30 days (`REFRESH_TOKEN_TTL`). See [security.md](./security.md) and `.env.example` for issuer, audience, secret, cookie, bcrypt, and throttling configuration.

## Campaigns (Phase 3A)

### Public discovery

`GET /api/v1/campaigns` and `GET /api/v1/campaigns/{campaignID}` are public. They expose only `ACTIVE` campaigns and return public fields: title, slug, description, brief, dates, thumbnail, platforms, and requirements. They never return a brand ID, budget, remaining budget, CPM, maximum payout, or private lifecycle data.

List parameters: `page`, `limit` (maximum 100), `search`, `platform`, `start_date`, `end_date`, `sort` (`created_at`, `start_date`, `end_date`, or `title`), and `direction` (`asc` or `desc`). Responses include `campaigns` and `pagination` with page, limit, total_items, and total_pages.

### Brand management

All brand routes require a valid active BRAND access token and active session:

- `POST /api/v1/campaigns` creates a DRAFT owned by the authenticated brand.
- `GET /api/v1/brand/campaigns` lists only that brand’s campaigns.
- `GET /api/v1/brand/campaigns/{campaignID}` gets a private campaign.
- `PATCH /api/v1/brand/campaigns/{campaignID}` edits only DRAFT or PAUSED campaigns.
- Lifecycle actions: `/publish`, `/pause`, `/resume`, `/complete`, and `/cancel`.

Create accepts decimal monetary values as strings, not floating-point values. `brand_id`, `remaining_budget`, status, timestamps, and financial history are never accepted from the client. Slugs are generated from the title when omitted and are immutable after creation.

### Administrative campaign access

Active ADMIN users may use `GET /api/v1/admin/campaigns`, `GET /api/v1/admin/campaigns/{campaignID}`, and the explicit `POST /api/v1/admin/campaigns/{campaignID}/cancel` operation. Administrative cancellation follows the same allowed non-terminal transition rules and produces a distinct audit action.

### Lifecycle

| From | Allowed action | To |
| --- | --- | --- |
| DRAFT | publish, cancel | ACTIVE, CANCELLED |
| ACTIVE | pause, complete, cancel | PAUSED, COMPLETED, CANCELLED |
| PAUSED | resume, complete, cancel | ACTIVE, COMPLETED, CANCELLED |
| COMPLETED/CANCELLED | none | terminal |

Publishing and resuming require valid campaign content, a usable positive remaining budget, an unexpired date range, and at least one allowed platform. Every create, update, transition, and administrative cancellation creates a safe audit record.

## Participation and submissions (Phase 3B)

All Phase 3B routes require a valid bearer token, an active account, and an active session. The server derives both the participant and submission owner from that token; client-provided ownership, status, moderation, metric, payout, and timestamp fields are ignored because they are not accepted by the request models.

### Participation

- `POST /api/v1/campaigns/{campaignID}/join` is available only to CLIPPER users. It creates an `ACCEPTED` membership for an ACTIVE campaign only after its start date and before its end date. A repeat request returns `409 CAMPAIGN_ALREADY_JOINED`; the database unique pair `(campaign_id, clipper_id)` remains the final duplicate guard.
- `GET /api/v1/clipper/campaigns` lists only the caller's memberships. `campaign_id`, participant `status`, `page`, `limit` (maximum 100), `sort` (`created_at`, `joined_at`, `status`), and `direction` (`asc`, `desc`) are supported.
- `GET /api/v1/clipper/campaigns/{campaignID}/participation` returns only the caller's membership.
- `GET /api/v1/brand/campaigns/{campaignID}/participants` is limited to the owning BRAND and supports `status`, pagination, and the same participation sort fields.
- `GET /api/v1/admin/participations` and `GET /api/v1/admin/participations/{participationID}` provide explicit ADMIN inspection only. They do not make an administrator a participant.

### Clip submissions

- `POST /api/v1/campaigns/{campaignID}/submissions` creates a `PENDING` submission for an accepted CLIPPER participant in an eligible ACTIVE campaign.
- `GET /api/v1/clipper/submissions`, `GET /api/v1/clipper/submissions/{submissionID}`, and `PATCH /api/v1/clipper/submissions/{submissionID}` are owner-only routes. A patch may contain only `platform`, `content_url`, and/or `caption`; at least one is required and the submission must still be `PENDING`.
- `GET /api/v1/brand/campaigns/{campaignID}/submissions` and `GET /api/v1/brand/campaigns/{campaignID}/submissions/{submissionID}` are restricted to the campaign-owning BRAND. A submission ID under an unrelated campaign path is hidden as not found.
- `GET /api/v1/admin/submissions` and `GET /api/v1/admin/submissions/{submissionID}` provide explicit ADMIN inspection. No approval, rejection, flagging, or other moderation route exists in this phase.

Submission lists accept `campaign_id`, `platform`, `status`, `page`, `limit` (maximum 100), `sort` (`submitted_at`, `created_at`, `updated_at`, `status`), and `direction` (`asc`, `desc`). Results use a stable ID secondary order and include `pagination` with `page`, `limit`, `total_items`, and `total_pages`.

Supported platforms are `TIKTOK`, `INSTAGRAM`, and `YOUTUBE`. URLs must be HTTPS, have no embedded credentials or port, match the selected platform's official host family, and contain a path. The API lowercases the host, removes query strings/fragments and trailing path slashes, and stores the resulting canonical URL. The global unique URL constraint rejects duplicate content after normalization. The API never fetches or scrapes submitted URLs.

Joining, creating a submission, and updating a submission write `campaign.joined`, `submission.created`, and `submission.updated` audit records respectively in the same transaction as the business write. Private ownership failures use a 404 response; missing authentication/session state uses 401 and pure role denials use 403. `CAMPAIGN_NOT_ACTIVE`, `CAMPAIGN_NOT_STARTED`, `CAMPAIGN_ENDED`, `SUBMISSION_REQUIRES_PARTICIPATION`, `SUBMISSION_DUPLICATE_URL`, `SUBMISSION_INVALID_URL`, `SUBMISSION_INVALID_PLATFORM`, `SUBMISSION_PLATFORM_NOT_ALLOWED`, and `SUBMISSION_NOT_EDITABLE` are the relevant Phase 3B domain errors.
