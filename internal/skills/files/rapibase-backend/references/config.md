# Instance configuration

`rapibase config get` and `rapibase config set KEY=VALUE [KEY=VALUE…]` (MCP: `get_config` and
`set_config`). Variables are MERGED with the ones already there; an empty value (`KEY=`) deletes one.
They exist so you don't have to stop and ask the person to go into the console to flip a flag.

The same variables as in the console are accepted (`SMTP_*`, `AUTH_*`, `CORS_ORIGINS`, `JWT_EXPIRY`,
`REFRESH_EXPIRY`, `OAUTH_*`). Any other is discarded and returned to you in `rejected`. `FN_*` secrets do
NOT go through here: the person loads them in the console (the project's Secrets tab), so third-party
keys never pass through the chat; `config get` shows them masked. Saving recreates the app container in
seconds and the data isn't touched. The dashboard's admin is NOT changed this way: that still requires a
session in the console.

```bash
rapibase config get
rapibase config set CORS_ORIGINS=<origin1>,<origin2>
rapibase config set AUTH_SIGNUP_ENABLED=true
```

`OAUTH_*` values are third-party credentials: the person pastes them in the console (Settings → Social
login), not in the chat ([auth.md](auth.md)).

Restarting the instance (`rapibase restart`) is almost never the fix: when it is, how to check first and
its rate limit are in the rapibase-observe skill (`references/config.md`).
