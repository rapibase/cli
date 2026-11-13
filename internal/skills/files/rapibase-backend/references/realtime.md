# Realtime — WebSocket

URL: `{base-url}/api/realtime/v1?apikey=<key>&token=<user-jwt>`, with the WebSocket scheme that matches
the instance's (`ws` for `http`, `wss` for `https`). The subprotocol `rapibase-realtime.v1+json` is
MANDATORY: `new WebSocket(url, ['rapibase-realtime.v1+json'])`. In the frontend the key is the ANON_KEY
(from the build's environment variable, never hand-written into the code).

```js
const url = RAPIBASE_URL.replace(/^http/, 'ws') + '/api/realtime/v1?apikey=' + RAPIBASE_ANON_KEY + '&token=' + jwt
const ws = new WebSocket(url, ['rapibase-realtime.v1+json'])
ws.onopen = () => ws.send(JSON.stringify(subscribeFrame))
```

```json
{"type":"subscribe","ref":"s1","channel":"room:42","config":{
  "postgres_changes":[{"event":"INSERT","schema":"public","table":"messages",
    "filter":{"op":"and","conditions":[{"column":"room_id","op":"eq","value":42}]}}],
  "broadcast":{"self":false},"presence":{"key":"user_7"}}}
```

THE JWT IS NOT OPTIONAL for listening to table changes: with the ANON_KEY the same rule as in REST
applies, the public key alone only reads what is public. Without a token only tables in RLS mode `public`
or `read_only` arrive, plus broadcast, presence and rpc (which don't touch the database). If you subscribe
and NOTHING arrives, check this before anything else. With the SERVICE_KEY everything arrives, but that
never goes in a frontend.

Incoming `postgres_changes` events carry the field `db_event` (INSERT|UPDATE|DELETE), `new` and `old`.
Other frames: broadcast, presence_track/untrack, rpc, heartbeat, set_auth.

Presence: `config.presence.key` only says which key identifies you in the channel; it does NOT mark you
as connected. To show up in the list of present users, after the subscribe send
`{"type":"presence_track","channel":"room:42","payload":{"name":"Ana"}}` (the payload is your state: the
others receive it). You get `presence_state` (the full list on joining) and `presence_diff`
(joins/leaves). To leave without disconnecting: `presence_untrack`; closing the socket leaves on its own.

Filters: structured JSON (eq neq lt lte gt gte in nin is like ilike contains contained_by match), not
PostgREST's DSL.

Is realtime delivering at all? `rapibase api get /api/realtime/metrics` (MCP: `api_get`; see the
rapibase-observe skill before restarting anything). Which rows each subscriber gets is decided by RLS: [rls.md](rls.md).
