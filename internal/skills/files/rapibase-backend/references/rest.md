# REST CRUD — /api/v1/rest/{table}

From the agent: `rapibase api` with the same path (MCP: the `api_*` tool of the method, `api_get`,
`api_post`…). Both use the service key, so they bypass RLS; `--anon` and `--jwt <token>` make
`rapibase api` call as a visitor or as a signed-in user instead ([credentials.md](credentials.md)).

```bash
# SELECT with filters: <column>.<op>=<value>  (op: eq neq gt gte lt lte like ilike is in)
rapibase api get "/api/v1/rest/products?page=1&page_size=50&order_by=created_at&order=desc&select=id,name,price&price.gt=100"
# → {"data":[...],"page":1,"page_size":50,"total_rows":N,"total_pages":N}   (no rows: "data":[])

rapibase api post /api/v1/rest/products --data '{"name":"X","price":9.99}'
rapibase api patch /api/v1/rest/products/1 --data '{"price":14.99}'
rapibase api delete /api/v1/rest/products/1

# what a visitor / a signed-in test user gets (RLS applies)
rapibase api get "/api/v1/rest/products?page_size=20" --anon
rapibase api get /api/v1/rest/orders --jwt <token>
```

From the app's code, the same paths with the app's keys:

```js
// visitor without a session: the ANON_KEY alone, GET only, and only tables in RLS mode public or read_only
const res = await fetch(`${RAPIBASE_URL}/api/v1/rest/products?page_size=20`, {
  headers: { apikey: RAPIBASE_ANON_KEY },
})

// logged-in user: ANON_KEY + the user's JWT (RLS applies)
await fetch(`${RAPIBASE_URL}/api/v1/rest/products`, {
  method: 'POST',
  headers: { apikey: RAPIBASE_ANON_KEY, Authorization: 'Bearer ' + jwt, 'Content-Type': 'application/json' },
  body: JSON.stringify({ name: 'X', price: 9.99 }),
})
// PATCH /api/v1/rest/products/1 and DELETE /api/v1/rest/products/1 work the same way
```

IMPORTANT: the filter format is `column.op=value` (e.g. `status.eq=active`) and ordering is
`order=asc|desc`. DON'T use `filter=col:op:val` or `order_dir`, even if they show up in old docs.
`page_size` max 1000. Views are read the same way.
