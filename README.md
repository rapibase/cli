# Rapibase CLI

The agent-native backend, from your terminal.

> **Status:** in development. Not published on npm yet.

```bash
npx rapibase@latest init
```

One command to log in (or create your account) in the browser, create or pick a project, link this
folder to it, write your `.env` and connect your coding agent (skills and MCP): Claude Code, Cursor,
OpenCode, Codex, Hermes or anything that reads `AGENTS.md`.

Then, from the same terminal (or install it with `npm i -g rapibase` and drop the `npx`):

```bash
rapibase deploy ./dist          # publish your frontend
rapibase env pull               # write the project keys to .env (values are never printed)
rapibase status                 # what is running and where
rapibase sql "select count(*) from orders"
rapibase logs --follow
```

The CLI talks to [Rapibase Cloud](https://rapibase.com). The Rapibase core is open source and can be
self-hosted; this CLI manages Cloud projects.

## Commands

| | |
|---|---|
| **Start** | `init`, `login`, `logout`, `whoami` |
| **Project** | `projects`, `projects create`, `link`, `unlink`, `status`, `env pull`, `open` |
| **Publish** | `deploy [dir]`, `deploys` (and `deploys activate` to roll back), `functions deploy\|list\|rm`, `services deploy\|update\|activate\|restart\|rm`, `logs` |
| **Operate** | `sql`, `api <get\|post\|put\|patch\|delete> <path>`, `rls`, `config get\|set`, `restart` |
| **Your agent** | `skill install\|update`, `mcp add` |

`rapibase <command> --help` shows the options and examples of each one, in English, Spanish or
Portuguese (`--lang`, `RAPIBASE_LANG` or your system language).

### Which project

Commands act on, in this order: `--project <slug|id>`, `RAPIBASE_PROJECT`, the nearest
`.rapibase/project.json` (written by `init` and `link`; no secrets, safe to commit; in a monorepo each app
can have its own), your only project, or a prompt. Commands that change something print the target
first (`→ my-app (acme)`).

## For coding agents and scripts

- `--json` prints exactly one object on stdout: `{"ok":true,"data":…}` or
  `{"ok":false,"error":{"code","message",…}}`. Progress goes to stderr.
- Without a TTY nothing prompts: a missing choice fails with `missing_flag` and the list of options, and
  nothing waits more than 90 seconds.
- `login` without a TTY prints the link and the code (`login_pending`, exit 10); run it again after the
  person approves.
- The agent never handles keys: `sql`, `api`, `functions` and `rls` go through Rapibase with the
  project's service key, which never leaves the server.

| Exit | `error.code` |
|---|---|
| 0 | — |
| 1 | `server_error`, `network`, and command errors (`sql_error`, `blocked`, `failed`) |
| 2 | `usage`, `missing_flag`, `not_linked`, `project_not_found` |
| 3 | `needs_login`, `session_expired` |
| 4 | `forbidden` (comes with `action_url`) |
| 5 | `plan_limit` |
| 6 | `project_not_running` |
| 7 | `rate_limited` (comes with `retry_after`) |
| 8 | `upgrade_required` |
| 10 | `login_pending`, `provisioning`: run the same command again |

## CI

Create a deploy token in the console (it only reaches deploys) and:

```yaml
- run: npx rapibase@latest deploy ./dist
  env:
    RAPIBASE_DEPLOY_TOKEN: ${{ secrets.RAPIBASE_DEPLOY_TOKEN }}
    RAPIBASE_PROJECT: ${{ vars.RAPIBASE_PROJECT }}   # the numeric project id
```

| Variable | |
|---|---|
| `RAPIBASE_TOKEN` | a connection token; takes priority over the saved login |
| `RAPIBASE_DEPLOY_TOKEN` | `rbdep_…`: only `deploy`, `deploys` and `services deploy` |
| `RAPIBASE_PROJECT` | project slug or id |
| `RAPIBASE_LANG` | `en`, `es` or `pt` |
| `RAPIBASE_NO_UPDATE_CHECK` | turns off the "new version" notice |
| `NO_COLOR` | plain output |

## Development

```bash
go test ./...
go run . --help
go run ./tools/syncskills -from ../rapibase-skills   # refresh the embedded skills
```

The skills come from [rapibase/skills](https://github.com/rapibase/skills), pinned in `skills.ref` and
embedded in the binary, so the CLI and its skills always match.

## Releasing

1. If the skills changed: bump their `metadata.version`, tag `rapibase/skills` and put that tag in
   `skills.ref` (CI fails if the embedded copy differs).
2. Push a `vX.Y.Z` tag here. The `release` workflow runs GoReleaser (binaries, checksums, GitHub
   Release) and publishes `rapibase` plus one `@rapibase/cli-<os>-<arch>` package per platform to npm,
   with provenance. A tag with a `-` (`v0.2.0-rc.1`) goes to the `next` dist-tag. Re-running a failed
   release is safe.

npm auth is trusted publishing (no secret), which can only be set on packages that already exist:

- **First release:** a granular npm token (read and write on `rapibase` and the `@rapibase` org,
  bypass 2FA, short expiry) as the `NPM_TOKEN` secret of the `NPM_TOKEN` environment of this repo
  (only `v*` tags, with a required reviewer).
- **Right before the second one:** on each of the 7 packages, Settings → Trusted publishing → GitHub
  Actions with `rapibase`, `cli`, `release.yml` and environment `NPM_TOKEN`, allowing `npm publish`. Then
  delete the token and the secret, and set Publishing access to "Require two-factor authentication and
  disallow tokens". A new trusted publisher expires if it does not publish within 2 days.

## License

MIT
