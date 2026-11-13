# Restarting the instance (and when not to)

`rapibase restart` (MCP: `restart_project`) restarts the project's app container; the database and
storage are untouched. Changing settings with `rapibase config set` (MCP: `set_config`) already recreates
the app container in seconds (rapibase-backend skill, `references/config.md`).

WHEN to restart: almost never. A restart doesn't fix code you wrote badly, and if something still fails
after restarting once, restarting again won't fix it either. Before asking for one, check that the
problem is the instance's and not what you wrote:

- Does the instance answer? `rapibase api get /api/v1/health` (MCP: `api_get /api/v1/health`).
- Is realtime delivering events? `rapibase api get /api/realtime/metrics` (MCP: `api_get`).
  If `realtime_wal_lag_bytes` doesn't show up, the replicator didn't start: that IS the instance. If it
  shows up and you still don't get changes, the problem is your subscription (start with the JWT:
  rapibase-backend skill, `references/realtime.md`).
- What do the logs say? `rapibase logs` (MCP: `get_logs`).

There is a limit of one restart every 3 minutes: if you get 429 with `retry_after_seconds` (CLI:
`rate_limited`, exit 7, with `retry_after`), wait that long instead of retrying in a loop.

An own service restarts on its own, without touching the instance: `rapibase services restart <name>`
(MCP: `restart_service`), same release and environment.
