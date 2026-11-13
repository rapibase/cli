# Automatic deploys with GitHub Actions (recommended)

GitHub builds; Rapibase only receives the artifact. Set it up only when the person asks for it. Once it
exists, deploying this repo = `git push` to that branch (see "When to deploy" in SKILL.md).

1. Ask the person to create a **deploy token** in the console (the project's Deploys tab;
   `rapibase open console`) and to store it as a secret themselves, without pasting it into the chat:
   `gh secret set RAPIBASE_DEPLOY_TOKEN` (gh asks for the value at the keyboard). The token only serves to
   publish (what it reaches: "Tokens" below): NEVER put the SERVICE_KEY in GitHub.

2. The project (not secret, you can set it yourself). With the deploy token it goes by its numeric id,
   the `project_id` in `.rapibase/project.json`. If that file is committed, the CLI already reads the id
   from it and this step is optional; when set, `RAPIBASE_PROJECT` wins over the file.

```bash
gh variable set RAPIBASE_PROJECT --body "<project_id from .rapibase/project.json>"
```

3. Create `.github/workflows/rapibase-deploy.yml` (the CLI recognizes this name: a manual
   `rapibase deploy` in this repo then warns that deploys go through `git push`):

```yaml
name: Deploy to Rapibase
on: { push: { branches: [main] } }
jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with: { node-version: 22, cache: npm }
      - run: npm ci && npm run build
      - name: Publish to Rapibase
        # in CI the CLI skips its "a workflow already deploys" and "uncommitted
        # changes" checks: this workflow is the deploy
        run: npx rapibase@latest deploy ./dist
        env:
          RAPIBASE_DEPLOY_TOKEN: ${{ secrets.RAPIBASE_DEPLOY_TOKEN }}
          RAPIBASE_PROJECT: ${{ vars.RAPIBASE_PROJECT }}
```

From then on: `git push` to main → build on GitHub → automatic deploy. Adjust `./dist` if your build goes
to another folder (e.g. `build` with CRA, `out` with Next export). The workflow needs no URL: the CLI
knows where Rapibase is, and the token plus `RAPIBASE_PROJECT` say which project.

## Services from CI

An own service ([services.md](services.md)) can be published from the same workflow, one step per
service, from its folder:

```yaml
      - name: Publish the api service
        run: npx rapibase@latest services deploy api ./api
        env:
          RAPIBASE_DEPLOY_TOKEN: ${{ secrets.RAPIBASE_DEPLOY_TOKEN }}
          RAPIBASE_PROJECT: ${{ vars.RAPIBASE_PROJECT }}
```

For a compiled service, build the Linux x86-64 binary in a previous step (see [services.md](services.md)).

## Tokens

- `RAPIBASE_DEPLOY_TOKEN` (deploy token, it starts with rbdep_) reaches `rapibase deploy`,
  `rapibase deploys` (list and `deploys activate`) and every `rapibase services …` command (list, deploy,
  activate, restart, update, rm). That covers the steps above.
- Anything else from CI (`rapibase functions deploy`, `sql`, `api`, `rls`, `config`, `logs`, `restart`,
  `tables`) needs `RAPIBASE_TOKEN`: a personal token (it starts with rbpat_) the person creates in the
  console (Agents → Tokens for servers and CI) and stores as a secret the same way, never through the
  chat. Add it only if the person wants that step in CI; the workflow above doesn't publish functions.

## An older workflow without the CLI

If the repo already has a workflow that uploads the build with a direct HTTP call instead of the CLI, it
keeps working: deploying is still `git push` to its branch. Don't rewrite it unless the person asks.
