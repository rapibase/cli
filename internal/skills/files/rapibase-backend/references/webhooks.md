# Webhooks (with conditions)

- Webhooks live in the instance dashboard → Webhooks (`rapibase open dashboard`). Events
  `INSERT:table` / `UPDATE:table` / `DELETE:table` or `TYPE:*`, payload
  `{id, event, table, timestamp, data, old_data}`, HMAC-SHA256 signature of the body in
  `X-Webhook-Signature` (`sha256=<hex>`). They fire from REST, the dashboard and the MCP; inserts made
  with direct SQL (`db.query` in functions, RPC, the SQL editor, `rapibase sql`, imports) do NOT fire
  webhooks.
- New users: the `INSERT:auth_users` event arrives with every signup (email, Google, Apple, GitHub,
  Facebook, native login or created through the API). `data` is the public user (id, email,
  email_verified, full_name, provider, created_at…; `provider` is email, google, apple, github or
  facebook), never the hash or tokens. `INSERT:*` does NOT include it: subscribe to `INSERT:auth_users`
  explicitly.
- Delivery and retries: the receiver must answer 2xx. If it fails (no connection, timeout, 408, 429 or
  5xx) it is retried at 1 min, 5 min, 30 min, 2 h and 6 h (6 attempts, persisted: they survive restarts);
  any other 4xx counts as failed with no retry. `X-Webhook-Id` (equal to the payload's `id`) repeats on
  every retry and on a manual resend: store it and discard the ones you already processed.
  `X-Webhook-Attempt` carries the attempt number. The dashboard's log shows the state and has "Resend now".
- CONDITIONS per webhook: a list of `{column, op, value}` (ops: eq neq gt gte lt lte contains is_null
  not_null), ANDed together and evaluated against the row: it is only delivered if ALL of them hold. They
  are set in the same dialog (or via `POST /api/dashboard/webhooks` with the `filter` field, which takes a
  dashboard admin session, not the service key). Use them to receive only what matters instead of
  filtering in the receiver.
- More complex logic (OR, calculations, querying other tables, calling another API): point the webhook's
  URL at a function of your own (`fn.http('name', handler, { public: true })` →
  `{base-url}/api/fn/name`, rapibase-functions skill), verify the `X-Webhook-Signature` there with the
  secret and write the logic in TypeScript. There is no code inside the webhook: conditions are only a
  filter.
- Push: see [push.md](push.md) (the full contract).
