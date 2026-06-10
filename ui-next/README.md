# MyPast Observer — `ui-next`

A modern rebuild of the MyPast Observer UI using **Next.js (App Router) + TypeScript + Tailwind + shadcn/ui**. It consumes the existing mypast Go API (`/api/v1/browse/*`).

This first cut ships the **Sessions** view as the reference pattern:

- Left **sidebar** navigation across the distillation pyramid (sessions → atoms → scenes → memories → pipeline → tasks). Sessions is live; the rest are scaffolded (marked `soon`).
- **Data table** (TanStack Table) with global **search**, column **sorting**, and **pagination**.
- Row click opens a **detail modal** showing the session's pipeline status, turns (with messages), atoms, and scenes.

## Two build modes

The client always talks to `/api/v1/*` on its own origin. How that origin reaches the real API differs by mode:

- **Dev** (`pnpm dev`): `src/app/api/v1/[...path]/route.ts` is a same-origin proxy that forwards `/api/v1/*` to `MYPAST_API_URL`, injecting optional Basic-auth credentials server-side (so they never reach the browser) and avoiding CORS.
- **Embed** (`pnpm build:embed`): a static export served by the Go binary under `/ui`. The Go server serves both the UI and `/api/v1/*`, so it's already same-origin — no proxy needed. The export drops the dev proxy route (a dynamic route handler can't be statically exported); see `scripts/build-embed.mjs`.

Other pieces:

- `src/lib/api.ts` / `src/lib/types.ts` — typed client (calls `/api/v1/*`) + response types.
- `src/lib/format.ts` — formatting helpers, including `pick()` which reads both `snake_case` (browse wrapper structs) and `PascalCase` (raw GORM models) keys.
- `src/components/app-sidebar.tsx`, `src/components/*` — UI; `/` renders the Overview.

## Configuration

Copy `.env.example` to `.env.local` and adjust:

```
MYPAST_API_URL=http://localhost:8080   # upstream API base (serves /api/v1/*)
MYPAST_API_USER=                       # optional Basic-auth user
MYPAST_API_PASS=                       # optional Basic-auth password
```

Point `MYPAST_API_URL` at a local Go binary (`http://localhost:8080`) or the
remote server (`https://mem.colinleefish.com`, which requires the Basic-auth
credentials). `.env.local` is gitignored.

## Develop

```bash
pnpm install
pnpm dev        # http://localhost:3000  (→ Overview)
```

Start the Go API separately (e.g. `docker compose up` in the repo root, or run the binary on `:8080`).

## Embed as the default UI

This UI is the one the Go binary serves at `/ui`. To rebuild and re-embed it:

```bash
pnpm build:embed
```

This static-exports the app (with `basePath: /ui`) and copies the output into `../internal/http/static/web/`, which `internal/http/static/static.go` embeds via `//go:embed all:web` (the `all:` prefix is required so the underscore-prefixed `_next/` asset dir is included). Rebuild the Go binary afterwards to pick up the new assets:

```bash
(cd .. && go build ./cmd/mypast)
```

The embedded `internal/http/static/web/` is generated output — regenerate it with `pnpm build:embed` rather than editing by hand.

## Verify

```bash
pnpm exec tsc --noEmit   # types
pnpm lint                # eslint
pnpm build:embed         # static export + embed into the Go binary
(cd .. && go test ./internal/http/static/)  # asserts the export is embedded
```
