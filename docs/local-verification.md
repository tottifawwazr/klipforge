# Local product-completion verification

Verified on 2026-11-06, Windows, branch `feat/product-completion`. Changes remain uncommitted and unpushed.

## Running preview

- Web: http://localhost:3002/login (production standalone build)
- API: http://127.0.0.1:8082/api/v1/health
- PostgreSQL: loopback 54329; development and test databases are separate.
- Redis: loopback 16379; integration tests use a separate logical database.
- `scripts/start-local.ps1` was executed successfully, applying zero pending migrations and validating the frontend/API bridge. It can restart the local development environment after the running processes are stopped. The current review server is the production build.

## Passing checks

- Go module verification, `go vet ./...`, full `go test -p 1 -count=1 ./...` with PostgreSQL/Redis integration enabled, and API build.
- Campaign integration tests rerun after normalizing empty platform/requirement arrays.
- Frontend Prettier, ESLint with zero warnings, TypeScript, and production build.
- Vitest: 4 files, 10 tests, including HttpOnly session handling, CSRF origin rejection, body limits, and empty logout responses.
- Playwright complete workflow: Brand creates/edits/publishes campaign; Clipper joins/submits/edits; Brand approves/completes; Admin previews/reserves/approves/records payout; Clipper reads notification. Also verifies restricted admin endpoints, cross-origin mutation rejection, token invisibility to JavaScript, and refresh recovery.
- Playwright responsive workflow: login/register plus all Brand, Clipper, and Admin tabs at 320, 390, 768, and 1440px; no page overflow after chart resizing settles and no browser exceptions. Final complete run: 2 browser workflows passed (53.6 seconds), including the hover/resize regression.
- `npm audit`: 0 vulnerabilities across production and development dependencies. Next.js upgraded to 16.4.5; minimatch is overridden specifically under the TypeScript parser to patched 11.0.5.
- `govulncheck`: 0 reachable vulnerabilities. It still reports non-reachable advisories in imported/required modules; this is not a claim that all upstream modules are advisory-free.
- CI workflow YAML parses successfully. Git diff whitespace check passes.

## Bugs resolved during verification

- Same-origin checks account for the public Host even when Next normalizes the internal request origin.
- Explicit form labels keep populated textarea/select controls accessible by their intended names.
- Logout preserves HTTP 204 and clears both cookies; failed logout requests show an error instead of throwing an unhandled browser rejection.
- Campaigns without requirements/platforms serialize empty arrays and render safely.
- Mobile workspace grid children can shrink. Chart tooltips are clipped within the chart so hidden tooltip coordinates cannot widen the page after hover and desktop-to-mobile resizing; browser regression checks cover hover and resizing.

## Not performed or not integrated

Docker image builds were not run because Docker is unavailable on this host. The portable local runtime and standalone web build were tested instead. GitHub Actions and its Linux race check have not run remotely. No production deployment, live social API requests, or actual money transfer was performed. Simulated metrics, manual payments, list caps, and production prerequisites are documented in README.

Browser workflow tests intentionally leave identifiable QA campaign/payment records in the local development database. Logs and screenshots are in ignored `.cache/data/` and `.codex/qa/preview/`; secrets are excluded from source control.
