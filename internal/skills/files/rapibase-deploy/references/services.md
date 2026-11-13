# Own services: any language (Go, Rust, Zig, Python, Node, Bun)

A service is a server of yours that runs next to the instance: a Go/Rust/Zig binary, a FastAPI API, an
Express app, a Next app with SSR… Use one when TS functions aren't enough or the person asks for it (see
"Before designing" in Setup); otherwise it isn't needed. **The same deploy rules apply**: only with an
explicit request.

- **Several per project, each in its own language** (Free 1, Pro 3, Team 4). Name: lowercase letters and
  digits, no dashes (`api`, `web`, `ml`, `worker`).
- **Where it answers:** on a route of the project's domain (by default `/<service>`; `/` to own the root),
  on its own subdomain `<service>-<project subdomain>`, on a custom domain ([domains.md](domains.md)) or on
  several at once. With an empty route and no subdomain or domain it is **internal only** (a worker that
  only the other services call, at `<service>:8080` over plain HTTP): `--path ""` (MCP: `path: ""`). The
  route changes without redeploying with `rapibase services update <name> --path …` (MCP:
  `update_service`; only when the person asks for it), or on the next `rapibase services deploy … --path`.
  The service receives the full path (mounted at `/api`, it gets `/api/orders`, not `/orders`). The
  instance's reserved routes never reach it.
- **It must listen on** `0.0.0.0:$PORT` (8080 today) and answer HTTP.
- **It receives in its environment:** `DATABASE_URL` (the project's Postgres over the internal network),
  `RAPIBASE_URL` (public URL), `RAPIBASE_INTERNAL_URL` (the instance without going out to the internet),
  `RAPIBASE_ANON_KEY`, `RAPIBASE_SERVICE_KEY`, `RAPIBASE_SERVICE` (its name), `RAPIBASE_BASE_PATH` (its
  route) and the `FN_*` secrets from the Secrets tab. **Don't hardcode keys.** For a new third-party
  secret, the person loads it in Secrets and the services are recreated on their own.
- **Compiled (Go, Rust, Zig, C):** compile on your machine for **Linux x86-64** and upload the binary
  (plus the files it reads). Nothing is compiled on Rapibase's side; the CLI checks the binary is a Linux
  x86-64 ELF before uploading and, for a macOS, Windows or arm64 binary, gives the right build command.
- **Interpreted (Python, Node, Bun):** upload the code with its lockfile, **without** `node_modules` or
  `.venv`; dependencies are installed over there (`uv`, `npm ci`, `pnpm`, `bun install`). TypeScript for
  Node: compile it first.
- **`rapibase.json` (optional, at the root of what you upload):**
  `{"runtime":"binary|python|node|bun","start":"...","path":"/api","subdomain":false}`.
  The runtime is detected automatically (Linux binary, `package.json` with a `start` script, `bun.lock`,
  `requirements.txt`…). **Python always needs** `start`, e.g.
  `uvicorn main:app --host 0.0.0.0 --port $PORT`. For binaries, `start` runs without a shell (no `$VARS`
  or pipes): the binary reads `PORT` from the environment.
- **Monorepo:** one deploy per service, from each one's folder.

## Publishing

`rapibase services deploy <name> [dir] [--binary <file>] [--path </prefix>] [--subdomain]`
`[--custom-domain <host>]` (MCP: `create_service_upload` with `service`, and `path` if it isn't
`/<service>`; it returns the build and upload command for your language: run it as given, its answer
carries the same fields as below). The service is created on its first deploy. Example in Go:

```bash
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o server .
rapibase services deploy api --binary ./server --path /api
# a binary that reads other files: put them in one folder with it and deploy the folder instead
```

Python, Node or Bun: `rapibase services deploy api ./api` (the folder with the code and its lockfile).

- The CLI answers when the service already responds (installing dependencies can take a few minutes):
  the service, release id, status `ready` and its URL. With `--json`:
  `{"ok":true,"data":{"service","release_id","status":"ready","url",…}}`: `url` when it is on a route,
  `subdomain_url` / `custom_domain_url` when it has them, `internal_url` when it is internal only, and
  `notes` with warnings.
- If the build or the start fails, the CLI prints the phase (`build` or `start`) and the log, and exits 1;
  with `--json`: `{"ok":false,"error":{"code":"failed","phase":"build|start","error":"…","log":"…",…}}`.
  Read the log, fix, and deploy again. The previous release keeps serving.
- The plan's service limit (Free 1, Pro 3, Team 4) comes back as `plan_limit`: tell the person; they
  decide (see Setup).
- `rapibase services` (MCP: `get_services`): route map, state of each service (running, asleep, crashed
  with an exit code or out of memory) and history with logs. `rapibase logs --service <name> [--follow]`
  (MCP: `get_logs` with `service`) for what it prints while running.
- Restart without redeploying (same release and environment): `rapibase services restart <name>` (MCP:
  `restart_service`).
- Rollback: `rapibase services activate <name> <release_id>` (MCP: `activate_service_release`); the
  release ids are in `rapibase services`. Change where it is served without redeploying:
  `rapibase services update <name> --path /v2` (also `--subdomain`, `--subdomain=false` to turn it off,
  and `--custom-domain`; MCP: `update_service`). Remove one: `rapibase services rm <name> --yes`, passing
  `--yes` only after the person said yes (MCP: `delete_service`); its route goes back to whatever is below.
  All of them only when the person asks.
- The last 5 releases are kept per service. They run without root, with RAM and CPU by plan (Free 128 MB,
  Pro 512 MB, Team 1 GB each). On the free plan they sleep after 15 minutes without requests and wake up
  with the next one (Go/Rust/Zig instantly; Python or Node, 1–3 s).
- From CI: [github-actions.md](github-actions.md).
