# Email (SMTP): what it covers and what it doesn't

The SMTP configured in the console (Settings → Email) is used ONLY for the app users' auth emails: email
verification, magic links and password reset. There is NO generic "send email" endpoint.

You can set `SMTP_HOST`, `SMTP_PORT`, `SMTP_USER` and `SMTP_FROM` with `rapibase config set` (MCP:
`set_config`; see [config.md](config.md)). `SMTP_PASS` is a third-party credential: the person enters it
(Settings → Email in the console, or running `rapibase config set SMTP_PASS=…` themselves); never type it
from the conversation.

For marketing, email notifications or any custom email: write a FUNCTION (rapibase-functions skill) that
calls your provider's HTTP API with `fetch` and the API key stored as an `FN_*` secret named after it
(`FN_<PROVIDER>_KEY`; the person creates it in the project's Secrets tab of the Rapibase Cloud console).
ANY provider with an HTTP API works — Resend, SendGrid, Mailgun, Postmark, Amazon SES, Brevo, Loops or
whichever the person already uses. The example is a generic sample: take the endpoint, the auth header
and the body fields from the provider's API docs.

```ts
const EMAIL_API = "<your provider's send-email endpoint, from its API docs>"

fn.job("send-email", (p) => {
  const r = fetch(EMAIL_API, {
    method: "POST",
    headers: { Authorization: "Bearer " + env.get("FN_<PROVIDER>_KEY") },   // the secret's real name
    body: { from: "<sender address on your domain>", to: (p as any).to, subject: (p as any).subject, html: (p as any).html },
  })
  if (!r.ok) throw new Error(r.body)
})
```
