# Automatic deploys with GitHub Actions (recommended)

GitHub builds; Rapibase only receives the artifact. Set it up only when the person asks for it. Once it
exists, deploying this repo = `git push` to that branch (see "When to deploy" in SKILL.md).

1. Ask the person to create a **deploy token** in the console (the project's Deploys tab;
   `rapibase open console`) and to store it as a secret themselves, without pasting it into the chat:
   `gh secret set RAPIBASE_DEPLOY_TOKEN` (gh asks for the value at the keyboard). The token only serves to
   publish: NEVER put the SERVICE_KEY in GitHub.

2. The project variable (not secret, you can set it yourself): the `project_id` from
   `.rapibase/project.json`.

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

The project's deploy token is accepted for service releases. If the CLI asks for a broader token, use
`RAPIBASE_TOKEN` instead: a server token the person creates in the console (Agents → Tokens for servers
and CI) and stores as a secret the same way. For a compiled service, build the Linux x86-64 binary in a
previous step (see [services.md](services.md)).
