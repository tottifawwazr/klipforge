# KlipForge Implementation Plan

## Scope and delivery rules

KlipForge will be built incrementally. Each phase must be formatted, linted, tested, and verified before the next phase begins. A feature is only marked complete after its runtime or automated verification passes.

Completed phases remain documented as verified baselines. The current product-completion branch continues the approved MVP through analytics, operations, and the complete UI. Live social metrics and payment transfers remain outside this MVP.

## Phase 1 — Foundation

Status: Complete (verified 2026-07-11)

- [x] Create the monorepo directory structure.
- [x] Add shared environment examples and local-development defaults.
- [x] Configure Docker Compose for PostgreSQL and Redis, including health checks and persistent volumes.
- [x] Initialize the Go API with configuration loading, structured logging, request IDs, secure baseline middleware, graceful shutdown, and a versioned health endpoint.
- [x] Initialize the Next.js App Router frontend with TypeScript, Tailwind CSS, ESLint, and Prettier.
- [x] Add a minimal professional landing page that displays live API connectivity.
- [x] Install dependencies and initialize local configuration.
- [x] Start PostgreSQL and Redis and confirm their health.
- [x] Start the Go API and Next.js application.
- [x] Verify the API health endpoint directly.
- [x] Verify frontend-to-API connectivity through the running frontend.
- [x] Run formatting, linting, tests, dependency auditing, and production builds; fix every discovered Phase 1 error.

### Phase 1 acceptance evidence

The applications are running through Docker Compose and all four services are reported healthy.

### Initialization and runtime

- `npx.cmd create-next-app@latest apps/web --typescript --tailwind --eslint --app --src-dir --use-npm --import-alias "@/*" --yes` — initialized Next.js 16.2.10, React 19.2.4, TypeScript, App Router, and Tailwind CSS.
- `npm.cmd install ...` — installed the required frontend libraries plus the formatting and Vitest toolchain; the lockfile is present.
- Official `golang:1.24-alpine` toolchain container: `go mod tidy` — resolved the Go module and generated `go.sum` without requiring a host Go installation.
- `docker compose --env-file .env.example config --quiet` — passed.
- `docker compose --env-file .env.example up -d postgres redis` — PostgreSQL and Redis started.
- `docker compose ... exec -T postgres pg_isready -U klipforge -d klipforge` — accepting connections.
- `docker compose ... exec -T redis redis-cli ping` — `PONG`.
- `docker compose --env-file .env.example up -d --build api web` — production API and web images built and started.
- Final `docker compose ps` — `postgres`, `redis`, `api`, and `web` all `healthy`.

### Backend quality checks

- `gofmt -l .` — no unformatted files.
- `go mod verify` — all modules verified.
- `go vet ./...` — passed.
- `go test -count=1 ./...` — passed across configuration, health, HTTP API, dependency, server, and command packages (14 top-level tests, with additional table-driven cases).
- Docker production build — passed as a non-root, statically compiled API image.

### Frontend quality checks

- `npm.cmd run format:check` — passed.
- `npm.cmd run lint` — passed with zero warnings.
- `npm.cmd run typecheck` — passed.
- `npm.cmd run test` — 2 test files passed, 4 tests passed.
- `npm.cmd run build` — Next.js production build passed; `/` is static and `/api/health` is dynamic.
- `npm.cmd audit --json --registry=https://registry.npmjs.org` — 0 known vulnerabilities after overriding Next.js's vulnerable transitive PostCSS version with 8.5.16.
- Docker production build — passed and reported 0 vulnerabilities during `npm ci`.

### Live verification

- `GET http://127.0.0.1:8080/api/v1/health` — HTTP 200, `status: ok`, with PostgreSQL and Redis both `up` and an `X-Request-ID` response header.
- `GET http://127.0.0.1:3000/api/health` — HTTP 200, `connected: true`; this Next.js server route reached and validated the Go health response and forwarded its request ID.
- `GET http://127.0.0.1:3000` — HTTP 200; rendered content contains the KlipForge brand and frontend/API health interface, with security headers present.

### Environment note

The host has Node.js but no Go or Make installation. Phase 1 Go initialization and quality checks therefore use the pinned official Go container. PowerShell's script policy blocks `npm.ps1`, so Windows verification uses `npm.cmd`. The main runtime remains the cross-platform Docker Compose workflow.

## Phase 2 — Data and identity

Status: Complete (verified 2026-07-11)

### Phase 2A — Schema, migrations, and development fixtures

Status: Complete (verified 2026-07-11)

- [x] Create ordered, reversible PostgreSQL migrations for identity, campaigns, submissions, metrics, payouts, notifications, audit logs, and idempotency keys.
- [x] Add normalized UUID-keyed tables, foreign keys, indexes, status checks, monetary checks, and timestamp triggers.
- [x] Add a Docker Compose migration runner with up, down, and status operations.
- [x] Add idempotent local-development seed data with bcrypt-hashed fixture credentials.
- [x] Document the schema, ERD, migration workflow, and local-only seed credentials.

### Phase 2B - Authentication and sessions

Status: Complete (verified 2026-07-11)

- [x] Short-lived JWT access tokens with strict claim, issuer, audience, algorithm, signature, and expiry validation.
- [x] Opaque hashed refresh tokens with transactional rotation, family/session tracking, and concurrent-refresh protection.
- [x] Refresh-token reuse detection, audit logging, session revocation, logout, and logout-all.
- [x] Registration, login, refresh, current-user, and authentication middleware endpoints.
- [x] Bcrypt passwords, HttpOnly refresh cookies, structured errors, and Redis-backed authentication rate limiting.

### Phase 2C - Authorization

Status: Complete (verified 2026-07-11)

- [x] Typed authenticated-principal context with JWT, account, and session identity.
- [x] Reusable authentication, active-state, role, any-role, policy, and ownership guards.
- [x] Explicit administrator exceptions and database-backed campaign, participation, submission, payout, and profile policies.
- [x] Safe 401, 403, and enumeration-resistant 404 authorization behavior with structured denial logging.
- [x] PostgreSQL ownership and horizontal privilege-escalation integration tests.

## Phase 3 — Campaign workflow

Status: Complete

### Phase 3A - Campaign management

Status: Complete (verified 2026-07-11)

- [x] Public campaign discovery with safe public representations, filtering, pagination, and allow-listed sorting.
- [x] Brand campaign create, read, update, platform/requirement replacement, and database-backed ownership enforcement.
- [x] Explicit publish, pause, resume, complete, cancel, and administrative cancellation transitions with audit history.
- [x] Campaign slug, brief, and thumbnail schema migration with deterministic seed compatibility.
- [x] PostgreSQL transaction, lifecycle, audit, and horizontal-access integration tests.

### Phase 3B - Participation and submissions

Status: Complete (verified 2026-07-12)

- [x] Campaign participation and self-service membership reads.
- [x] Clip submission creation, ownership reads, safe PENDING updates, URL validation, and audit logging.
- [x] Brand and explicit admin inspection routes with PostgreSQL ownership checks.
- [x] Migration `000007_submission_caption`, deterministic seed, rollback, PostgreSQL integration, API build, and live endpoint verification.

### Phase 3C - Submission moderation

Status: Complete (verified 2026-07-12)

- [x] Brand-owned and administrative moderation queues with safe filters, search, pagination, and sorting.
- [x] Explicit approval, rejection, flagging, and flagged-resolution transitions.
- [x] Reviewer identity, timestamps, normalized reasons, and transactional audit logging.
- [x] PostgreSQL row-lock concurrency protection and ownership integration tests.
- [x] Reversible migration `000008_submission_moderation_constraints`, seed compatibility, API build, and live endpoint verification.

## Phase 4 - Metrics and analytics

Status: Implemented; local verification documented separately

- Deterministic, explicitly simulated metrics for supported social URLs.
- Redis-coordinated worker, idempotent snapshots, monotonic counters, transactional state rechecks.
- Role-scoped aggregates, 30-day timeline, generation-aware cache, and Redis leaderboards.

## Phase 5 - Payouts and operations

Status: Implemented; local verification documented separately

- Decimal CPM calculations with per-clip cap and remaining-budget protection.
- Idempotent creation, row-lock concurrency, preview/revalidation, failure refunds, and retry reservations.
- Audited approval and external payment recording; no real money transfer.
- Role-scoped payout history, owned notifications, and administrative audit view.

## Phase 6 - Product UI

Status: Implemented; local verification documented separately

- Responsive Brand, Clipper, and Admin workspace with authentication and secure cookie bridge.
- Campaign lifecycle/editor, participation, pending submission editing, and moderation.
- Analytics, rankings, payouts, notifications, audit, loading/error/empty states, confirmations.
- Current browsing caps and provider limitations documented in README.

## Phase 7 - Quality and documentation

Status: Local MVP validation complete; remote CI not run

- Go unit/integration tests, frontend unit tests, and Playwright campaign-to-payout workflow.
- CI definition with ephemeral PostgreSQL and Redis, production build, race checks, and browser tests.
- Local startup helper, security/API documentation, and preview instructions.
- No commit, push, or deployment is part of the current local validation.
