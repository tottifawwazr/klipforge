# KlipForge architecture

## Current services

KlipForge is a Docker Compose monorepo with these runtime services:

- `apps/web`: Next.js App Router frontend.
- `services/api`: Go HTTP API using `pgx` for PostgreSQL and Redis for health checks and authentication throttling.
- `postgres`: PostgreSQL 16 persistent local database.
- `redis`: Redis 7 persistent local cache and coordination foundation.

The API now exposes Phase 2B authentication under `/api/v1/auth` while retaining the Phase 1 health endpoints. Product and authorization handlers remain unimplemented.

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
- Role claims are identity context only in Phase 2B. Phase 2C must add explicit backend authorization and must not rely on frontend role visibility.

## Authentication request flow

```text
HTTP handler -> Redis rate limit -> authentication service
                                  -> bcrypt / JWT / opaque-token service
                                  -> PostgreSQL repository transaction
                                  -> audit_logs

Bearer middleware -> validate JWT -> validate active user + session -> request context
```

Refresh rotation locks the presented token row and performs replacement, revocation, and audit insertion in one PostgreSQL transaction. Session IDs remain stable across rotations; token family IDs provide lineage. See [`security.md`](./security.md) for the threat behavior and [`api.md`](./api.md) for the transport contract.

## Operational workflow

Use Docker Compose tool-profile commands or the matching Make targets:

- `migrate up` applies pending migrations.
- `migrate down` rolls back the latest migration.
- `migrate status` reports applied and pending files.
- `seed` upserts local-development fixtures.
- `db-reset` removes local Compose volumes before migrating and seeding from scratch.

See [`database-schema.md`](./database-schema.md) for the ERD, database constraints, fixture contents, and PowerShell-safe commands.
