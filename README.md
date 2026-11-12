# Rapibase CLI

The agent-native backend, from your terminal.

> **Status:** in development. Not published on npm yet.

```bash
npx rapibase@latest init
```

One command to log in (or create your account) in the browser, link this folder to a Rapibase project,
write your `.env` and connect your coding agent: Claude Code, Cursor, OpenCode, Codex or Hermes.

Then, from the same terminal (or install it with `npm i -g rapibase` and drop the `npx`):

```bash
npx rapibase deploy ./dist   # publish your frontend
npx rapibase env pull        # write the project keys to .env (never printed)
npx rapibase status          # what is running and where
```

Every command has `--json` output and a non-interactive mode, so coding agents can run it too.

The CLI talks to [Rapibase Cloud](https://rapibase.com). The Rapibase core is open source and can be
self-hosted; this CLI manages Cloud projects.

## License

MIT
