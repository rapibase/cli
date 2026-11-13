# Domains: the project's subdomain, custom domains and service hosts

Change domains only when the person asks for it. The project's custom domain is set in the console
(`rapibase open console`); a service's with `rapibase services update <name> --custom-domain <host>` or
`--subdomain` (MCP: `update_service`).

## Where things answer

- **The project** answers on its subdomain on Rapibase Cloud (`rapibase status --json` →
  `data.project.app_url`; `rapibase open app`). The site, the instance and the services mounted on routes
  all share it (reserved routes: see Setup).
- **A service with `subdomain: true`** also answers on its own host, `<service>-<project subdomain>`: the
  whole host is the service's, with no reserved routes there (handy for a separate public API).
- **Custom domains** can be put on the project (the site and everything on its routes) or on a single
  service (e.g. `api.<your-domain>` for the `api` service). A domain belongs to one project or one service.

## Setting them

- **The project's subdomain or custom domain:** in the console (`rapibase open console` → the project's
  domain settings). Give the person the exact hostname to enter. A domain under Rapibase Cloud's own
  domain is refused (use your own).
- **A service's custom domain:** `rapibase services update <name> --custom-domain <hostname>` (a full
  hostname like `api.<your-domain>`; `--custom-domain ""` removes it), or `--custom-domain` on
  `rapibase services deploy` — ONLY if the person gave the domain. `services update` also switches
  `--subdomain` (`--subdomain=false` turns it off) and `--path`; only the flags you pass change. MCP:
  `update_service` with `custom_domain`, `subdomain` and `path`, or `create_service_upload` with
  `custom_domain`. The resulting URLs come from `rapibase services` (MCP: `get_services`).

## DNS: what the person adds at their DNS provider

The answer of the call (its `dns` field) says exactly which record to create. The rules:

- **A subdomain** (`shop.<your-domain>`, `api.<your-domain>`): a CNAME to the project's subdomain.
- **A root domain** (`<your-domain>` itself) can't carry a CNAME: an ALIAS/ANAME (or CNAME flattening) to
  the same target or, if the provider doesn't have that, an A record to the IP the answer gives. Also
  delete the redirect or parking page the registrar usually puts on `@`.
- HTTPS is issued on the first visit once DNS points correctly; there is nothing else to configure.
- DNS can take a while to propagate. If it doesn't work, check the record against the answer character by
  character before changing anything else.

## After moving to a custom domain

- Social login: the instance dashboard → Auth → Providers shows the exact callback URLs to register with
  each provider (`rapibase open dashboard`); check them against the providers' settings after a domain
  change (rapibase-backend skill, `references/auth.md`).
- `redirect_url` for social login must be on the app's domain (`APP_URL`, `AUTH_REDIRECT_URL` or an origin
  in `CORS_ORIGINS`): update them with `rapibase config set` if the app's domain changes.
