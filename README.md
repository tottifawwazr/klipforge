# KlipForge

KlipForge is a creator-campaign and short-form content analytics platform for brands, clippers, and platform administrators. The repository includes the Phase 1 foundation, Phase 2A database schema/fixtures, and Phase 2B backend authentication.

## Stack

- Next.js with TypeScript, App Router, and Tailwind CSS in `apps/web`
- Go HTTP API in `services/api`
- PostgreSQL 16 and Redis 7
- Docker Compose for a repeatable local environment

Backend role/ownership authorization and product features remain intentionally deferred.

## Prerequisites

- Docker Desktop with Docker Compose v2
- Optional for running outside containers: Go and Node.js/npm versions compatible with each application
- `make` is optional; every Make target is a thin wrapper around the documented commands

## Configuration

Copy the development example before changing defaults:

```powershell
Copy-Item .env.example .env
```

The checked-in values are local-development placeholders only. The important public endpoints are:

| Service | Default address |
| --- | --- |
| Web | `http://localhost:3000` |
| API | `http://localhost:8080` |
| API health | `http://localhost:8080/api/v1/health` |
| Auth API | `http://localhost:8080/api/v1/auth` |
| PostgreSQL | `localhost:5432` |
| Redis | `localhost:6379` |

## Run locally

Start only the infrastructure:

```bash
docker compose up -d postgres redis
docker compose ps
```

Build and start the full stack:

```bash
docker compose up --build
```

In another terminal, verify the API:

```bash
curl http://localhost:8080/api/v1/health
```

Then open `http://localhost:3000`; the landing page performs a real API connectivity check.

Apply the schema and local-only development fixtures before using authentication:

```powershell
docker compose --env-file .env.example --profile tools run --rm migrate up
docker compose --env-file .env.example --profile tools run --rm seed
```

The authentication variables in `.env.example` are safe local placeholders. Production requires independently generated `JWT_SECRET` and `REFRESH_TOKEN_PEPPER`; token lifetimes, bcrypt cost, cookie settings, and Redis rate limits are also configurable there. See `docs/api.md` and `docs/security.md`.

Verify the frontend-to-API bridge without a browser:

```bash
curl http://localhost:3000/api/health
```

A passing response contains `"connected":true` and the validated Go API health payload.

Stop the stack without deleting PostgreSQL or Redis data:

```bash
docker compose down
```

## Quality commands

```bash
make format
make lint
make test
make build
```

Equivalent application-specific commands can be run from `services/api` and `apps/web`. Migration and seed commands are available through the Make targets and Docker Compose `tools` profile.

## Current limitations and next work

Phase 2C will add backend-enforced role and ownership authorization. Authentication frontend pages are also intentionally absent. See `docs/implementation-plan.md` for the phased delivery plan.
