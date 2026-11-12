# RLS (row level security)

Modes per table: `owner` (each user sees only their rows, via owner_column = auth.uid()), `authenticated`,
`public`, `read_only`, `custom`. `owner`, `authenticated` and `public` ALWAYS let logged-in users write
(applying them gives that permission back even if it was removed before). For data everyone reads and
only the server writes (catalogs, prices, configuration) use `read_only`: only the SERVICE_KEY, the
dashboard and the functions write it. With the SERVICE_KEY, RLS does NOT apply: that's why it never goes
in a frontend. `auth.uid()` is available in policy SQL.

## Setting a table's mode

- **CLI** (the agent does it):
  - `rapibase rls` lists the tables with a managed mode; `rapibase rls list <table>` shows the mode, the
    owner column and the policies of one table.
  - `rapibase rls set <table> <mode>`, and for `owner` also `--owner-column <col>` (a uuid column; it is
    defaulted to `auth.uid()` on insert, so inserts don't need to send it). Idempotent: running it again
    switches the mode.
  - `rapibase rls disable <table> --yes` removes the protection. Ask the person first: every key reaches
    every row again.
- **MCP:** the same with `list_rls`, `set_rls` (`table`, `mode`, `owner_column`) and `disable_rls`.
- **Instance dashboard** (the person does it): `rapibase open dashboard` → the table → RLS.
- Don't emulate a mode with your own `GRANT` / `CREATE POLICY` statements: the mode is recorded in an
  internal `_rapibase_*` table that raw SQL can't write, and realtime and `rapibase rls` read the recorded
  mode, so they would ignore yours.
- `custom` mode = RLS on with NO policies (default-deny) until you add your own with `rapibase sql`
  (MCP: `execute_sql`), e.g. `CREATE POLICY ... USING (auth.uid() = user_id)`. Read the policies of a table
  with `rapibase sql 'SELECT policyname, cmd, roles, qual, with_check FROM pg_policies WHERE tablename = $1' --params '["orders"]'`.

## What each role sees

REST, RPC and realtime apply the SAME RLS (realtime, event by event and per subscriber):

| Table mode | anon (no JWT) | authenticated user | SERVICE_KEY |
|---|---|---|---|
| *not configured* | ✗ | ✓ | ✓ |
| public | ✓ | ✓ | ✓ |
| read_only | ✓ read only | ✓ read only | ✓ |
| authenticated | ✗ | ✓ | ✓ |
| owner | ✗ | only their rows | ✓ |
| custom | ✗ | ✗ | ✓ |

- A newly created table is NOT public: it reaches logged-in users, not anonymous ones.
- `public` and `read_only` are the deliberate way out for feeds without login (order tracking,
  scoreboards).
- `custom` is NOT delivered through realtime except with the SERVICE_KEY: its policies are SQL that
  Postgres evaluates and realtime can't run them. If you need such a table live, use a view in mode
  owner/public, or send the change yourself with broadcast from a function.
- An RLS change takes up to 30 s to reach subscribers that are already connected.
- `rapibase api` acts with the service key, so it sees everything: test what users see from the app.
