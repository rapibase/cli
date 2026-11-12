# Email (SMTP): what it covers and what it doesn't

The SMTP configured in the console (Settings → Email; or the `SMTP_*` variables with `rapibase config
set`, see [config.md](config.md)) is used ONLY for the app users' auth emails: email verification, magic
links and password reset. There is NO generic "send email" endpoint.

For marketing, email notifications or any custom email: write a FUNCTION (rapibase-functions skill) that
calls your provider's HTTP API with `fetch` and the API key stored as an `FN_*` secret (Secrets tab of the
project in the Rapibase Cloud console; the person creates it there). ANY provider with an HTTP API works —
Resend, SendGrid, Mailgun, Postmark, Amazon SES, Brevo, Loops or whichever the person already uses. The
example has Resend's shape only as a sample:

```ts
const EMAIL_API = "<your provider's send-email endpoint, from its API docs>"

fn.job("send-email", (p) => {
  const r = fetch(EMAIL_API, {
    method: "POST",
    headers: { Authorization: "Bearer " + env.get("FN_RESEND_KEY") },
    body: { from: "<sender address on your domain>", to: (p as any).to, subject: (p as any).subject, html: (p as any).html },
  })
  if (!r.ok) throw new Error(r.body)
})
```
