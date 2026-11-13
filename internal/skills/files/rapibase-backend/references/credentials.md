# How to work: CLI, MCP, credentials and the API's auth rules

## CLI first, MCP as fallback, keys never in the chat

The main path is the **Rapibase CLI**. `rapibase whoami` shows the session and `rapibase status` the
linked project. Admin work goes through `rapibase sql` and `rapibase api`, which add the project's service
key on the server side: the agent never sees a key.

The **Rapibase MCP** is the fallback (OAuth: the person approves in the browser, there are no keys to
copy). `rapibase init` registers it, or `rapibase mcp add [--agent …] [--scope …]`; a newly registered MCP
usually shows up only after the agent restarts. An MCP pinned to a project acts on that project when you
don't pass `project`; the other projects the person authorized stay reachable with `list_projects`, but
use them only when the person asks for them by name.

- `rapibase projects` / `rapibase status` (MCP: `list_projects` / `get_project`): projects, their mode
  (read/write), their URL and the console. If there are several and the person didn't say which one,
  ASK before acting.
- If a command or tool answers with an `action_url` (missing write permission, paused project…), give the
  person that link and retry when they confirm. `rapibase open connections` opens the console page where
  the person decides which projects each agent connection reaches and whether it can write.
- For DDL (create/drop) and anything destructive, ask for confirmation.
- With a session but no linked folder, `rapibase link <slug>` is lighter than `init`: it only writes
  `.rapibase/project.json`. Without a TTY, `init` needs `--project <slug>` (an existing project) or
  `--name <name>` (a new one); otherwise it answers `missing_flag` with the list of projects.
- `rapibase skill uninstall [--agent …] [--scope project|user]` and
  `rapibase mcp remove [name] [--agent …] [--scope project|user]` undo `skill install` and `mcp add`,
  only when the person asks (`--scope local` is the same as `project`: this repo).

If there is neither a CLI session nor the MCP, ask the person to run `npx rapibase@latest init` or to
connect the MCP (the project's Connect tab in the console) instead of asking for keys. On a server or in
CI without a browser, OAuth is replaced by a **server token** the person creates in the console (Agents →
Tokens for servers and CI) and leaves in the environment variable `RAPIBASE_TOKEN`: the CLI reads it, and
an MCP config (pointed at the account MCP address that the console's Connect tab shows) references it as
the header `Authorization: Bearer ${RAPIBASE_TOKEN}`. Never ask them to paste that token to you.

Tokens in CI, and what each one reaches:

- `RAPIBASE_DEPLOY_TOKEN` (a project's deploy token, it starts with rbdep_; the project's Deploys tab):
  `rapibase deploy`, `rapibase deploys` (list and `deploys activate`) and every `rapibase services …`
  command (list, deploy, activate, restart, update, rm).
- `RAPIBASE_TOKEN` (a personal token, it starts with rbpat_; Agents → Tokens for servers and CI) is needed
  for `functions`, `sql`, `api`, `rls`, `config`, `logs`, `restart` and `tables`.
- With the deploy token the project goes by its numeric id (`--project <id>` or `RAPIBASE_PROJECT=<id>`).
  A committed `.rapibase/project.json` already carries the id, so with that file nothing extra is needed.
  `RAPIBASE_PROJECT` and `--project` win over `.rapibase/project.json`.

### SQL from the agent

`rapibase sql '<SQL>'`, `rapibase sql --file schema.sql` or `rapibase sql -` (the SQL from stdin) (MCP:
`execute_sql`) runs SQL on the project's database:

- Parametrized with `$1`, `$2`… and `--params '<json array>'`, or `--param <value>` repeated in order. Use
  single quotes in the shell so `$1` isn't expanded:
  `rapibase sql 'SELECT * FROM orders WHERE id = $1' --params '[42]'`.
- 60 s per call by default, up to 600 with `--timeout <seconds>` (MCP: `timeout_seconds`).
- Several statements separated by `;` take no params and return no rows: the SELECT goes in its own call.
- Without `--read` it needs write permission, even for a SELECT: on a project or connection in read mode
  it answers `forbidden`.
- To only read, add `--read` (MCP: `query_sql`): `rapibase sql --read 'SELECT …'`. It runs ONE statement
  as a database user that can only read, inside a read-only transaction, so it works on a read-only
  connection too. It sees every row (RLS doesn't filter it, like the service key); anything that writes,
  DDL and several statements in one call fail.

Inspecting the schema (these work with read-only connections too):

```bash
rapibase tables          # every table: row count, primary key and columns (MCP: list_tables)
rapibase tables orders   # one table: columns, types, nullability, defaults, primary key, foreign keys (MCP: describe_table)
rapibase rls list orders # its RLS mode, owner column and policies (MCP: list_rls)
# indexes aren't in `rapibase tables`: read them with sql --read (MCP: query_sql)
rapibase sql --read 'SELECT indexname, indexdef FROM pg_indexes WHERE tablename = $1' --params '["orders"]'
```

Extensions (MCP: `list_extensions`): PostGIS, pg_trgm, pg_cron, `vector` and others are available. List
them with `rapibase sql --read 'SELECT name, installed_version FROM pg_available_extensions ORDER BY name'` and
enable one with `CREATE EXTENSION IF NOT EXISTS <name>`.

### MCP tools and their CLI equivalent

| MCP tool | CLI |
|---|---|
| `list_projects`, `get_project` | `rapibase projects`, `rapibase status` |
| `create_project` | `rapibase projects create --name <name>` (a free project, or one covered by the person's Team plan), or `rapibase init --name <name>` |
| `execute_sql` | `rapibase sql` |
| `query_sql` (read only) | `rapibase sql --read` |
| `list_tables`, `describe_table` | `rapibase tables`, `rapibase tables <name>` |
| `create_table`, `drop_table` | `rapibase sql 'CREATE TABLE …'` / `'DROP TABLE …'` (confirm first) |
| `query_rows`, `insert_row`, `update_row`, `delete_row` | `rapibase api get` / `post` / `patch` / `delete` on `/api/v1/rest/{table}[/{id}]` |
| `list_extensions` | `rapibase sql --read` on `pg_available_extensions` |
| `list_rls`, `set_rls`, `disable_rls` | `rapibase rls list [table]`, `rapibase rls set <table> <mode>`, `rapibase rls disable <table> --yes` after the person said yes ([rls.md](rls.md)) |
| `list_buckets`, `create_bucket`, `update_bucket`, `delete_bucket` | `rapibase api` on `/api/v1/storage/buckets[/{name}]` |
| `list_objects`, `delete_object`, `create_signed_url` | `rapibase api` on `/api/v1/storage/…` ([storage.md](storage.md)) |
| `upload_object`, `download_object` | `rapibase api post …/upload -F file=@<file>`, `rapibase api get …/<key> -o <file>` ([storage.md](storage.md)) |
| `list_storage_rules`, `set_storage_rule`, `delete_storage_rule` | `rapibase api` on `/api/v1/storage/rules` |
| `api_get`, `api_post`, `api_update` (PATCH or PUT), `api_delete` | `rapibase api get` / `post` / `patch` / `put` / `delete <path>` |
| `deploy_functions` | `rapibase functions deploy [dir]` |
| `get_config`, `set_config` | `rapibase config get`, `rapibase config set KEY=VALUE` |
| `restart_project` | `rapibase restart` |
| `get_logs` | `rapibase logs [--service <name> \| --container db\|storage] [--follow]` |
| `list_deploys`, `create_site_upload`, `activate_deploy` | `rapibase deploys`, `rapibase deploy [dir]`, `rapibase deploys activate <release_id>` |
| `get_services`, `create_service_upload`, `restart_service`, `delete_service` | `rapibase services`, `rapibase services deploy <name> [dir]`, `rapibase services restart <name>`, `rapibase services rm <name> --yes` |
| `update_service`, `activate_service_release` | `rapibase services update <name> --path …`, `rapibase services activate <name> <release_id>` |
| — (CLI only) | `rapibase api get <path> --anon`, `rapibase api <method> <path> --jwt <token>` (below) |
| — (CLI only) | `rapibase skill uninstall [--scope …]`, `rapibase mcp remove [name] [--agent …] [--scope …]` |

```
rapibase api <get|post|patch|put|delete> <path> [--data '<json>' | --data @file.json | --data - | -F field=@file]
             [--content-type <type>] [-o <file>] [--anon | --jwt <token>]
```

It calls any path under `/api/` of the instance, query string included (`--data -` reads the body from
stdin; `--content-type` overrides the guessed type). Quote paths that carry `?` or `&`.

**`rapibase api` and the `api_*` tools act with the service key: they bypass RLS.** To check what a
visitor or a user sees, use `rapibase api` as them (the CLI never prints the key; the MCP has no such mode):

- Visitor without a session (anon key only): `rapibase api get /api/v1/rest/<table> --anon`.
- Signed-in user (anon key + that user's JWT): sign a test user in with
  `rapibase api post /api/v1/auth/signin --anon -d '{"email":"…","password":"…"}'` (or `signup` to create
  one), take `token` from the answer and pass it on: `rapibase api get /api/v1/rest/<table> --jwt <token>`.
  Use a test account, never a real person's password. The token expires (1 h by default): sign in again.

## Credentials: the .env and nothing else

When the app's code or a script needs keys:

1. Run `rapibase env pull`. It writes or merges `.env` with `RAPIBASE_URL`, `RAPIBASE_ANON_KEY`,
   `RAPIBASE_SERVICE_KEY`, `RAPIBASE_DB_URL` and `RAPIBASE_PROJECT_ID` (other variables stay as they
   are), adds `.env` to `.gitignore` before writing, writes `.env.example` with the names, and never
   prints values. If `.env` is already tracked by git it writes nothing and explains how to take it out
   of the repo.
   - `--framework vite|next|astro|expo|sveltekit` also adds the public variables with the framework's
     prefix (`VITE_RAPIBASE_URL`, `NEXT_PUBLIC_RAPIBASE_ANON_KEY`…). The service key and the database URL
     never get a public prefix: the CLI refuses even if asked.
   - `--file <path>` writes another file (`.env.local`; or `.env.staging` together with
     `--project <staging project>` when the person names that project).
2. Without the CLI (MCP only): write `.env.example` (it is committed):
   ```
   # Copy the values from the console: <console_url from get_project> → ".env file" card
   RAPIBASE_URL=<api_url from get_project>
   RAPIBASE_PROJECT_ID=<the id at the end of console_url>
   RAPIBASE_ANON_KEY=
   RAPIBASE_SERVICE_KEY=
   RAPIBASE_DB_URL=
   ```
   Make sure `.env` is in `.gitignore`, and ask the person to copy the block from the console's ".env
   file" card and paste it into `.env`.
3. **Never read, print or echo `.env` or its variables**, and never ask the person to paste a key into
   the chat. Code and scripts load the file themselves (for example `node --env-file=.env scripts/seed.js`);
   your own admin calls go through `rapibase api` and `rapibase sql`.

In these skills, `{anon-key}` = `RAPIBASE_ANON_KEY`, `{service-key}` = `RAPIBASE_SERVICE_KEY` and
`{base-url}` = `RAPIBASE_URL`. HTTP examples show what the app's code sends (method, path, headers, body);
when the frontend is hosted by the project it shares the API's origin and can use relative paths
(`fetch('/api/v1/rest/...')`).

- ANON_KEY: public, goes in the frontend (from the build's environment variable, never hand-written into
  the code).
- SERVICE_KEY: SECRET, full access and bypasses RLS. Backend and scripts only. Never in the frontend.

## Auth rules on the whole /api/v1/* API

- Header `apikey: <key>` ALWAYS required.
- With the ANON_KEY, `Authorization: Bearer <user-jwt>` is also required. The only exception is visitors
  without a session (apps you can browse without signing up): with the ANON_KEY alone you can GET
  `/api/v1/rest/{table}` on tables in RLS mode `public` or `read_only`, and POST `/api/v1/rpc/{fn}` on
  functions with `GRANT EXECUTE ON FUNCTION fn TO rapibase_anon`. Everything else without a JWT gives 401.
  An expired or invalid token gives 401 (it is not treated as a visitor): renew it.
- With the SERVICE_KEY no JWT is needed.
- Errors: `{"error":"message"}`. Health: `GET /api/v1/health` (`rapibase api get /api/v1/health`, MCP:
  `api_get`).
- `_rapibase_*` tables are internal: never touch them (blocked even in raw SQL).

Recommended flow: list the tables (`rapibase tables`) → describe the table (`rapibase tables <name>`) →
operate.
