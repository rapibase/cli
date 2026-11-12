# Functions runtime: credentials, secrets and limits

## Credentials and secrets inside a function

- The instance's OWN keys are already built in: `env.get("SERVICE_KEY")`, `env.get("ANON_KEY")` and
  `env.get("APP_URL")`. Use them to call the same instance's API with `fetch` (push, storage, auth
  admin…). NEVER write a hardcoded key in a functions file, and don't duplicate the service key as an
  `FN_` secret: you already have it.

```ts
fn.job("notify-user", (p) => {
  const r = fetch(env.get("APP_URL") + "/api/v1/push/send", {
    method: "POST",
    headers: { apikey: env.get("SERVICE_KEY") },
    body: { user_id: (p as any).user_id, title: "Notice", body: (p as any).text },
  })
  if (!r.ok) throw new Error(r.body)
})
```

- `FN_*` secrets: ONLY for THIRD-PARTY credentials (Stripe, Resend, FCM…). The PERSON creates them in the
  Rapibase Cloud console (the project's Secrets tab), so the third party's key never passes through the
  chat; you only tell them the name your code expects (e.g. `FN_RESEND_KEY`). Read them with
  `env.get("FN_X")`. Never put them in the frontend. (`rapibase config set` doesn't take `FN_*`.)

## Limits and runtime behavior (avoid discovering them the hard way)

- No npm, no imports: each file is self-contained and only the host APIs exist (`db`, `fetch`, `env`,
  `jobs`, `console`/`log`, `sleep`). There is no filesystem, `require` or Node. If the job needs npm
  packages, that's a reason for an own service (rapibase-deploy skill), not a workaround here.
- HARD timeouts: 30 s per HTTP invocation; 5 min for cron and jobs. `fetch` is synchronous (15 s by
  default, `timeoutMs` max 60000, response capped at 5 MB). `sleep(ms)` max 10 s.
- Workers: 3 attempts with backoff (30 s × attempt number); when they run out, the job stays `failed`
  with `last_error` visible in `GET /api/v1/jobs` (`rapibase api get /api/v1/jobs`).
- Cron: standard 5-field spec (no seconds), in SERVER time (UTC on Rapibase Cloud). An invalid spec is
  reported as a load error; it doesn't fail silently.
- `console.log` and execution errors go to the instance's logs: `rapibase logs` (MCP: `get_logs`; or the
  Logs tab in the console). It is your debugging tool.
- Upload: max 512 KB per file and 50 files per request; file names `[a-zA-Z0-9._-]` with a `.ts`/`.js`
  extension; function/job/cron names `[a-z0-9_-]` (max 64, lowercase).
- Don't return pending promises: there is no event loop. An async handler works only because `await` on
  the host's synchronous APIs resolves instantly.
- Direct SQL from a function (`db.query`, `db.exec`) doesn't fire webhooks (rapibase-backend skill,
  `references/webhooks.md`).
