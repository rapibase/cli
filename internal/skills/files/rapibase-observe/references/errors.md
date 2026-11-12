# Errors: what they mean and what to do

## General API errors

A route that doesn't exist under `/api/…` answers **404** JSON `{"error":"route not found","method","path"}`,
never the dashboard's HTML. If you see HTML, the base URL is wrong.

Every `/api/v1/*` error is JSON `{"error":"message"}` (some areas add fields, below).

## CLI exit codes

With `--json` the error is `{"ok":false,"error":{"code","message",…}}`. Every error says what happened,
what to do and where.

| Exit | `error.code` | When | What to do |
|---|---|---|---|
| 0 | — | ok | |
| 1 | `server_error`, `network` | unexpected error or no connection | retry once; if it persists, `rapibase status` and `rapibase logs` |
| 2 | `usage`, `missing_flag`, `not_linked`, `project_not_found` | wrong usage, missing data, or the linked project no longer exists | the message lists the missing flags and the possible values (e.g. the projects). `not_linked`: `rapibase link <slug>` or `init`, asking which project if there are several; `project_not_found`: `rapibase link` |
| 3 | `needs_login`, `session_expired` | no session, or the refresh expired | `npx rapibase@latest init --agent <agent> --json` (or `rapibase login`) and show the person the link |
| 4 | `forbidden` | the connection doesn't reach the project or it is read-only for it | give the person the `action_url`; retry when they confirm |
| 5 | `plan_limit` | plan limit (free project already used, services, size) | tell the person; they decide. If the free project is already used, the error names it: use that project, or the person deletes it from the console. Don't push a paid plan |
| 6 | `project_not_running` | instance paused, provisioning or in error; carries `status` | provisioning: `rapibase status --wait`. Paused: the person resumes it from the console (`action_url`, or `rapibase open console`) |
| 7 | `rate_limited` | carries `retry_after` (e.g. one restart every 3 minutes) | wait that long instead of retrying in a loop |
| 8 | `upgrade_required` | this CLI version is no longer compatible | run it as `npx rapibase@latest <command>` (a global install: `npm i -g rapibase`) |
| 10 | `login_pending`, `provisioning` | pending state | `login_pending`: show `verification_uri_complete` and `user_code`, wait for the approval, run the same command again. `provisioning`: run the same command again |

MCP tools fail the same way: an error text and, when the person has to act, an `action_url`.

## By area

- **Auth (401):** missing `apikey`; or ANON_KEY without `Authorization: Bearer <jwt>` outside the visitor
  cases (GET on `public`/`read_only` tables, RPCs granted to `rapibase_anon`); or an expired or invalid
  token, which is NOT treated as a visitor: refresh it with `POST /api/v1/auth/token`.
- **Signup 403 `Public signup is disabled`:** someone set `AUTH_SIGNUP_ENABLED=false`; fix it with
  `rapibase config set AUTH_SIGNUP_ENABLED=true`, never with a SERVICE_KEY workaround.
- **Native login:** 401 with the reason (signature, audience, expired, nonce), 404 if the provider isn't
  configured, 502 if the provider's keys couldn't be fetched; 400 for a token without an email on the first
  login (request the `email` scope).
- **Social login on the web:** "redirect_uri_mismatch" at the provider = the callback URI doesn't match
  character by character (https, subdomain, no trailing slash). `redirect_url` must be on the app's domain
  (`APP_URL`, `AUTH_REDIRECT_URL` or `CORS_ORIGINS`; localhost is fine in development).
- **REST:** filters are `column.op=value` and ordering `order=asc|desc`; `filter=col:op:val` or
  `order_dir` from old docs don't work. `page_size` max 1000.
- **RPC:** JSON with `error`, `code`, `message` and, if the function set them, `detail` and `hint`. Status:
  403 without permission (42501), 404 nonexistent function, 409 duplicate key (23505), 504 timeout, 400 the
  rest (RAISE included). A visitor calling a function not granted to `rapibase_anon`: 401.
- **Storage:** 413 or 415 when an upload breaks the bucket's `file_size_limit` or `allowed_mime_types`
  (with the reason); 409 when `upsert=false` and the file exists.
- **Realtime: subscribed and nothing arrives:** check the JWT first (without it only `public`/`read_only`
  tables arrive). Tables in `custom` mode aren't delivered (only with the SERVICE_KEY). An RLS change takes
  up to 30 s to reach connected subscribers. Is realtime running at all: `rapibase api get
  /api/realtime/metrics` ([config.md](config.md)).
- **Push:** a missing or invalid field is **400** with `field`; an unknown `user_id` is 404; sending
  another user's `user_id` with a JWT is 403. In `/push/send` results, `errors` lists up to 50 provider
  rejections; only tokens the provider declares dead are removed (`removed:true`), any other error (bad
  credentials, invalid payload, network) keeps the token: read the `error`. `devices:0` = the user has no
  device registered.
- **SQL (`rapibase sql`, MCP `execute_sql`):** the Postgres error comes back as the error message.
  `_rapibase_*` tables are blocked. Several statements separated by `;` return no rows. Calls stop at 60 s
  by default (`--timeout` up to 600).
- **Functions:** compile errors show up on `rapibase functions deploy` as `file:line` (that file is
  skipped; the others stay loaded). Runtime errors and `console.log` are in `rapibase logs`. Hard timeouts:
  30 s per HTTP call, 5 min for cron and jobs. A job that used its 3 attempts stays `failed` with
  `last_error` in `rapibase api get /api/v1/jobs`. An invalid cron spec is a load error.
- **Services:** a failed deploy says its phase (`build` or `start`) and carries the log; the previous
  release keeps serving. `rapibase services` shows a crashed service with its exit code or out of memory;
  `rapibase logs --service <name>` shows what it printed. A macOS, Windows or arm64 binary is refused: build
  for Linux x86-64.
- **Site deploys:** the CLI refuses a folder with `.env*`, `*.pem`, `*.key`, `.git` or `node_modules`,
  without `index.html`, over 50 MB zipped / 200 MB extracted / 3000 files, with a dirty git tree without a
  TTY (`--allow-dirty`), or in a repo that deploys with GitHub Actions (`--force`; see the rapibase-deploy
  skill before using it).
- **Webhooks:** failed deliveries (anything but 2xx) are retried at 1 min, 5 min, 30 min, 2 h and 6 h; a
  4xx other than 408/429 is not retried. The instance dashboard's Webhooks log shows each one and has
  "Resend now". Inserts made with direct SQL don't fire webhooks.
- **Config:** `rapibase config set` returns variables it doesn't accept in `rejected`.
- **Restart:** 429 with `retry_after_seconds` (CLI: `rate_limited`, exit 7): once every 3 minutes.
