# KlipForge

KlipForge is a creator-campaign and short-form content analytics platform for brands, clippers, and platform administrators. This repository currently contains **Phase 1 only**: the monorepo foundation, local infrastructure, a Go API skeleton, and a Next.js web skeleton.

## Phase 1 stack

- Next.js with TypeScript, App Router, and Tailwind CSS in `apps/web`
- Go HTTP API in `services/api`
- PostgreSQL 16 and Redis 7
- Docker Compose for a repeatable local environment

Database migrations, seed data, authentication, authorization, and product features intentionally begin in later phases.

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
| PostgreSQL | `localhost:5432` |
| Redis | `localhost:6379` |

## Run Phase 1

Start only the infrastructure:

```bash
docker compose up -d postgres redis
docker compose ps
```

Build and start the full Phase 1 stack:

```bash
docker compose up --build
```

In another terminal, verify the API:

```bash
curl http://localhost:8080/api/v1/health
```

Then open `http://localhost:3000`; the landing page performs a real API connectivity check.

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

Equivalent application-specific commands can be run from `services/api` and `apps/web`. Migration and seed targets deliberately fail with a Phase 2 message so they cannot imply nonexistent functionality.

## Current limitations and next work

Phase 2 will add the normalized database schema, migrations, development seed data, JWT authentication, refresh-token rotation, and backend-enforced role authorization. See `docs/implementation-plan.md` for the phased delivery plan.
