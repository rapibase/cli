# Before designing: how an app is laid out on Rapibase

**Choose what best solves what the person asked for, not the biggest thing.** This guide describes
options; it doesn't ask you to use any one in particular. For the vast majority of apps the simplest is
best: the project's database, auth and storage, a static frontend and **TypeScript functions** for the
server logic. All of that fits the free plan, is cheap to run and is what any model knows best. Don't
assume a paid plan or several services are needed: build the app with the plan the project has and add
pieces only when there is a concrete reason (below: when a service makes sense). If something doesn't fit
the current plan, tell the person and let them decide.

Read this BEFORE choosing routes or the shape of the backend. Everything lives on **one domain** (the
project's, or its custom domain), with no CORS between the parts:

| Route | Served by |
|---|---|
| `/api/v1`, `/api/dashboard`, `/api/realtime`, `/api/fn`, `/mcp`, `/_` | **the Rapibase instance (reserved)** |
| each own service's route (`/api`, `/ml`…) | that service, in whatever language |
| `/` | the static site (frontend) or the service mounted at `/` |

- **Never use the reserved routes in your code.** Everything else under `/api` is yours: `/api/orders`,
  your own `/api/auth/...` or `/api/v2` yes; `/api/v1/...` no (it belongs to the instance).
- **Default option: TS functions** (`/api/fn/<name>`, rapibase-functions skill): endpoints, cron, webhooks
  and workers, with nothing extra to deploy. They cover most backends.
- **A service** (rapibase-deploy skill, `references/services.md`) makes sense when there is a concrete
  reason: the backend already exists or the person wants it in another language (Go, Python, Rust…), a
  library that only exists in that ecosystem or npm packages are needed, a framework with SSR (Next, Nuxt,
  SvelteKit), websockets or long-running processes of your own. The free plan already includes one. A
  project can have several (Free 1, Pro 3, Team 4), each in its own language; they talk to each other at
  `<service>:8080` (plain HTTP on the internal network) and to the instance through
  `RAPIBASE_INTERNAL_URL`, without going out to the internet.
- Examples:
  - **SPA (Vite/React) + Go API:** site at `/` + service `api` at `/api`. The frontend calls
    `fetch('/api/orders')`.
  - **SaaS with a TS frontend + Go API + Python worker:** site at `/`, service `api` at `/api`, service
    `ml` at `/ml` (or internal only: the Go service calls it at `ml:8080`).
  - **Next.js / Nuxt / SvelteKit with SSR:** service `web` at `/` (Node or Bun). Its route handlers under
    `/api/...` work, except under the reserved routes.
  - **Separate public API:** the service on its own subdomain (`subdomain: true` → host
    `<service>-<project subdomain>`), where there are no reserved routes:
    `rapibase services update <name> --subdomain` (or `--subdomain` on `rapibase services deploy`; MCP:
    `update_service`). Its exact URL comes from `rapibase services` (`subdomain_url`); never build it by
    hand.
- `rapibase services` (MCP: `get_services`) shows the project's current route map: look at it before
  choosing.
