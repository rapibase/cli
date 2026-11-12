# RPC — Postgres functions

```bash
# the JSON body maps to the function's named arguments
rapibase api post /api/v1/rpc/top_products --data '{"p_limit":10}'
```

The app sends the same thing: `POST /api/v1/rpc/top_products` with `apikey` (and the user's
`Authorization: Bearer <jwt>`) and the JSON body `{"p_limit":10}`.

Create the function with `rapibase sql` (MCP: `execute_sql`) or the dashboard's SQL editor. It is the
Rapibase way of doing "functions" as logic next to the data, with no deploy (TypeScript functions are the
rapibase-functions skill).
It answers an array of rows; with no rows it answers `[]`.
If the function fails (RAISE EXCEPTION, constraint, permissions) it answers JSON with the parts separately:

```json
{"error":"insufficient balance","code":"P0001","message":"insufficient balance","detail":"...","hint":"..."}
```

`detail` and `hint` only come if the function set them
(`RAISE EXCEPTION '...' USING DETAIL = '...', HINT = '...'`). Status: 403 without permission (42501), 404 nonexistent function, 409 duplicate key
(23505), 504 timeout, 400 the rest (RAISE included). To tell your own errors apart in the app, use
`USING ERRCODE = 'P0001'` or a code of your own and read `code`.

From the app: with ANON_KEY + JWT, the function runs as the user (with RLS). For a visitor WITHOUT a
session to call it, it has to be opened explicitly; otherwise it gives 401:

```sql
GRANT EXECUTE ON FUNCTION top_products(int) TO rapibase_anon;
```

It runs as `rapibase_anon`: it only sees tables in mode `public` or `read_only` (unless you declare it
`SECURITY DEFINER`, which runs with the owner's permissions: then you filter what it returns yourself).
