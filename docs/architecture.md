# KlipForge architecture

## Current services

KlipForge is a Docker Compose monorepo with these runtime services:

- `apps/web`: Next.js App Router frontend.
- `services/api`: Go HTTP API using `pgx` for PostgreSQL and Redis for health checks and authentication throttling.
- `postgres`: PostgreSQL 16 persistent local database.
- `redis`: Redis 7 persistent local cache and coordination foundation.

The API exposes authentication under `/api/v1/auth` while retaining the public health endpoints. Phase 2C adds reusable authorization guards and domain policy code, but intentionally exposes no incomplete product or test-only routes.

## Database architecture

PostgreSQL owns durable relational data. The normalized schema separates identity, campaign configuration, participant submissions, metrics, financial records, notifications, audit history, and future idempotency records.

The Go `klipforge-migrate` command applies ordered SQL files from `/migrations` and records each successful transaction in `schema_migrations`. The same API image also contains the `klipforge-seed` command for repeatable local fixtures.

```text
Docker Compose tools profile
  migrate service -> PostgreSQL + read-only /migrations mount
  seed service    -> PostgreSQL

API runtime
  Go API          -> PostgreSQL (pgx pool)
  Go API          -> Redis (health + authentication rate limits)
```

The Compose `migrate` and `seed` services are intentionally one-shot tools. They do not expose HTTP ports, alter API routing, or become long-running application services.

## Data boundaries

- PostgreSQL stores the source-of-truth relational records and durable audit trail.
- Redis is not a system of record and remains outside Phase 2A schema work.
- Password values are stored only as bcrypt hashes in PostgreSQL fixtures; no secret values are added to `.env.example`.
- Authentication uses thin HTTP handlers, an authentication service, password/token services, a PostgreSQL repository, and request-context middleware.
- Authentication produces a typed safe principal. Authorization consumes that principal through reusable role/state guards and PostgreSQL-backed domain policies; frontend role visibility is never a security boundary.

## Authentication request flow

```text
HTTP handler -> Redis rate limit -> authentication service
                                  -> bcrypt / JWT / opaque-token service
                                  -> PostgreSQL repository transaction
                                  -> audit_logs

Bearer middleware -> validate JWT -> validate active user + session -> request context
Request context   -> role/state guard -> ownership policy -> future product handler
```

Refresh rotation locks the presented token row and performs replacement, revocation, and audit insertion in one PostgreSQL transaction. Session IDs remain stable across rotations; token family IDs provide lineage. See [`security.md`](./security.md) for the threat behavior and [`api.md`](./api.md) for the transport contract.

## Middleware order

Global middleware runs in this order: request ID, structured request logging, panic recovery, security headers, CORS, request-size limit, and request timeout. Route-local authentication then validates the bearer token and current database state, attaches the typed principal, and composes active-user, active-session, role, and resource-policy guards as required.

Public health, registration, login, refresh, and cookie-based logout routes remain public by design. `/auth/me` and `/auth/logout-all` require authentication plus current account and session state. Product handlers added in later phases must use the policy guard rather than manually comparing route or request-body owner IDs.

## Authorization package

`internal/authorization` contains parameterized ownership repositories and policies for campaigns, participation, submissions, payouts, and profiles. Each policy declares its own administrator exception; no global administrator bypass exists. Inaccessible owned resources map to a safe not-found decision, while infrastructure errors remain internal.

## Campaign module

Phase 3A adds `internal/campaign` with separate HTTP handler, service, repository, validation, lifecycle, and audit responsibilities. The campaign service owns the DRAFT/ACTIVE/PAUSED/COMPLETED/CANCELLED transition rules. Its repository uses PostgreSQL transactions for campaign records, platform and requirement replacement, and audit entries.

Public campaign routes mount separately from brand and administrator management routes. Brand and administrator routes compose the existing authentication, active-user, active-session, and role middleware; the campaign service then verifies database ownership before returning private data or writing state.

## Operational workflow

Use Docker Compose tool-profile commands or the matching Make targets:

- `migrate up` applies pending migrations.
- `migrate down` rolls back the latest migration.
- `migrate status` reports applied and pending files.
- `seed` upserts local-development fixtures.
- `db-reset` removes local Compose volumes before migrating and seeding from scratch.

See [`database-schema.md`](./database-schema.md) for the ERD, database constraints, fixture contents, and PowerShell-safe commands.
