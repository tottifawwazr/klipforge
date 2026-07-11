# KlipForge Web

The Phase 1 KlipForge frontend is a Next.js App Router application with strict TypeScript, Tailwind CSS, TanStack Query, and a server-side bridge to the Go API health endpoint.

## Local development

Requirements:

- Node.js 20.9 or newer
- The KlipForge API available at `http://127.0.0.1:8080`, or `API_INTERNAL_URL` set to another base URL

Install and start from this directory:

```powershell
npm.cmd ci
npm.cmd run dev
```

Open `http://localhost:3000`. The browser queries `/api/health`; that Next.js route calls `/api/v1/health` on the Go API and validates the response before displaying the live status.

## Quality commands

```powershell
npm.cmd run format:check
npm.cmd run lint
npm.cmd run typecheck
npm.cmd run test
npm.cmd run build
```

The production image is built by the root Docker Compose workflow. `API_INTERNAL_URL=http://api:8080` is supplied at container runtime and is never included in the browser bundle.
