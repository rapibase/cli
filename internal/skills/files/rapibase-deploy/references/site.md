# Publishing the frontend (built-in hosting)

Only after an explicit request from the person (see "When to deploy" in SKILL.md).

The project serves your frontend at its URL, the SAME origin as the API. In the frontend's code use
relative paths (`fetch('/api/v1/rest/...')`): zero CORS, zero URL environment variables.

Publishing a build (Vite, Next export, any static site):

```bash
npm run build
rapibase deploy            # or: rapibase deploy ./dist
# → prints the site's URL and the release id; the release is live at once
```

- **Folder** (when you don't pass one): vite/astro → `dist`; next with `output: 'export'` → `out`; CRA →
  `build`; SvelteKit with adapter-static → `build`; Nuxt generate → `.output/public`; an `index.html` at the
  root → `.`. If the folder doesn't exist, the CLI says which build command to run.
- **Next.js with SSR** is not a static site: the CLI detects it and suggests
  `rapibase services deploy web --path /` ([services.md](services.md)).
- **Checks before uploading:** zip ≤ 50 MB, extracted ≤ 200 MB, ≤ 3000 files, and an `index.html`.
- **It never publishes secrets** (the site is public): if the folder has `.env*`, `*.pem`, `*.key`, `.git`
  or `node_modules`, it aborts and names the files. `--exclude <glob>` leaves files out of the zip.
- **Guards:** with a dirty git tree it aborts when there is no TTY, unless `--allow-dirty` (use it when the
  person asked to publish what is there: don't demand commits). If
  `.github/workflows/rapibase-deploy.yml` exists, this repo publishes with `git push`; without a TTY it
  aborts unless `--force`, which you only pass when the person explicitly wants a manual deploy anyway.
- Each upload creates a release and activates it (the last 10 are kept).
- List releases: `rapibase deploys` (MCP: `list_deploys`). One-step rollback, only when the person asks:
  `rapibase deploys activate <release_id>` (MCP: `activate_deploy`).
- SPA fallback: paths without an extension serve `index.html` (React Router and the like work).
- The instance's admin dashboard is always available at `{base-url}/_/` (`rapibase open dashboard`): don't
  use `/_` as a route of your app.
- The site's URL: `rapibase status --json` → `data.project.app_url` (MCP: `get_project`), or
  `rapibase open app`.
- Staging: each environment is its own project. `rapibase deploy --project <staging project>` only when
  the person names that project.
- MCP fallback: `create_site_upload` returns a one-time upload URL (valid 10 minutes) and the exact
  command that sends the zip of the build; the answer is `201 {"release_id":"...","url":"..."}`. The zip
  holds the build folder's contents, with `index.html` at its root: adjust the folder in the command if
  the build isn't in `dist`. Windows has no `zip`: use the answer's `command_windows` (it zips with the
  `tar` that comes with Windows).

## Testing is not deploying

To try changes, use the frontend's local environment (`npm run dev`). If it needs the project's API under
the same origin, Vite's proxy can point at `RAPIBASE_URL`:

```js
// vite.config.js
import { defineConfig, loadEnv } from 'vite'

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')   // reads .env without printing it
  return { server: { proxy: { '/api': { target: env.RAPIBASE_URL, changeOrigin: true, ws: true } } } }
})
```
