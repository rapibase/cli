# Instance configuration

`rapibase config get` and `rapibase config set KEY=VALUE [KEY=VALUE…]` (MCP: `get_config` and
`set_config`). Variables are MERGED with the ones already there; an empty value (`KEY=`) deletes one.
They exist so you don't have to stop and ask the person to go into the console to flip a flag.

The same variables as in the console are accepted (`SMTP_*`, `AUTH_*`, `CORS_ORIGINS`, `JWT_EXPIRY`,
`REFRESH_EXPIRY`, `OAUTH_*`, and `FN_*` secrets). Any other is discarded and returned to you in
`rejected`. `config get` shows secrets masked. Saving recreates the app container in seconds and the data
isn't touched. The dashboard's admin is NOT changed this way: that still requires a session in the
console.

Third-party secrets (`FN_*`, `SMTP_PASS`, the `OAUTH_*` client secrets) are entered by the person: in the
console (the project's Secrets tab, Settings → Email, Settings → Social login) or by running
`rapibase config set` themselves. Never type one from the conversation; tell them the name your code
expects.

```bash
rapibase config get
rapibase config set CORS_ORIGINS=<origin1>,<origin2>
rapibase config set AUTH_SIGNUP_ENABLED=true
```

For `OAUTH_*`, the console's Settings → Social login is where the person pastes them ([auth.md](auth.md)).

Restarting the instance (`rapibase restart`, MCP: `restart_project`) is almost never the fix: when it is,
how to check first and its rate limit are in the rapibase-observe skill (`references/restart.md`).
