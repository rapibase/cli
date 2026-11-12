# App users' auth — /api/v1/auth/*

Public signup comes OPEN in Rapibase Cloud projects. If signup still answers
`403 {"error":"Public signup is disabled"}`, someone set `AUTH_SIGNUP_ENABLED=false` on this project: don't
retry it or build a workaround with the SERVICE_KEY. Fix it with `rapibase config set AUTH_SIGNUP_ENABLED=true`
(MCP: `set_config`; see [config.md](config.md)).

## Email and password

What the app sends (`apikey: {anon-key}`, `Content-Type: application/json`):

```
POST /api/v1/auth/signup   {"email":"…","password":"…","full_name":"Ada"}
→ 201 {"token":"...","refresh_token":"...","expires_in":3600,"user":{...}}   (the field is "token", not "access_token")

POST /api/v1/auth/signin   {"email":"…","password":"…"}

POST /api/v1/auth/token    {"refresh_token":"..."}   (refresh; rotating: it invalidates the previous refresh_token)
```

```js
const r = await fetch(`${RAPIBASE_URL}/api/v1/auth/signin`, {
  method: 'POST',
  headers: { apikey: RAPIBASE_ANON_KEY, 'Content-Type': 'application/json' },
  body: JSON.stringify({ email, password }),
})
const { token, refresh_token, user } = await r.json()   // then: Authorization: 'Bearer ' + token
```

Others: `/auth/signout`, `/auth/magiclink`, `/auth/resend`, `/auth/verify`, `/auth/forgot-password`,
`/auth/reset-password`.

User management (service key): `GET` / `POST /api/v1/auth/users`, `GET` / `PUT` / `DELETE
/api/v1/auth/users/:id` (GET → `{"user":{...}}`, 404 if it doesn't exist), `POST /api/v1/auth/users/bulk`.
From the agent: `rapibase api get /api/v1/auth/users`, `rapibase api post /api/v1/auth/users --data '{…}'`,
and so on (MCP: `api_get` / `api_post` / `api_update` / `api_delete`).

## Social login on the web (Google, GitHub, Facebook)

1. The owner creates the OAuth app in the provider's console (step by step in "Creating the credentials",
   below) with this callback URL: `{base-url}/api/v1/auth/oauth/{provider}/callback` (provider: google |
   github | facebook).
2. They paste the credentials in the Rapibase Cloud console → Settings → Social login
   (`OAUTH_GOOGLE_CLIENT_ID`/`_SECRET`, etc.). A provider becomes active once its pair is complete. These
   are third-party credentials: the person pastes them in the console, not in the chat.
3. In the frontend, the login button is a plain link (no SDKs). `redirect_url` is absolute and on the
   app's domain:

```js
const redirect = encodeURIComponent(location.origin + '/auth/callback')
googleLink.href = `/api/v1/auth/oauth/google?redirect_url=${redirect}`   // prefix RAPIBASE_URL if the frontend is hosted elsewhere
```

4. After authorizing, the instance creates (or finds by email, already verified) the user and redirects to
   `redirect_url` with `#access_token=...&refresh_token=...&expires_in=...&type=oauth&provider=google`: the
   SAME format as the magic link, so the frontend's callback serves both.

- `GET /api/v1/auth/oauth/providers` → `{"providers":[...], "native":[...]}`: `providers` lists the
  configured redirect providers and `native` the ones that accept an id_token (`google`, `apple`). Use it
  to render only the buttons that work.
- `redirect_url` must be on the app's domain (`APP_URL`, `AUTH_REDIRECT_URL` or an origin listed in
  `CORS_ORIGINS`; localhost is always accepted in development). It only accepts http(s): in mobile apps use
  the native login below.
- The owner sees each provider's state (active or not, missing variables, exact URLs to copy) in the
  instance dashboard → Auth → Providers (`rapibase open dashboard`).

## Native login in mobile apps: Google and Sign in with Apple

In Flutter, iOS or Android the app does NOT use the redirect: it signs in with the provider's SDK, gets an
`id_token` and exchanges it for a Rapibase session. Rapibase verifies the signature with the provider's
public keys, the issuer, and that the token was issued for one of the project's client IDs. No secret is
needed.

1. Configuration (the person sets it in Rapibase Cloud → Settings → Social login):
   - Google: `OAUTH_GOOGLE_CLIENT_IDS` = the Android and/or iOS client IDs, comma-separated.
     `OAUTH_GOOGLE_CLIENT_ID` (the web one) is accepted too: on Android, `google_sign_in` issues the
     id_token for the `serverClientId`, which must be that web client ID.
   - Apple: `OAUTH_APPLE_CLIENT_IDS` = the app's bundle ID (iOS/macOS) and, for the web with Apple JS
     (`usePopup: true`), the Services ID, comma-separated. "Sign in with Apple" must be enabled on the App ID.
2. Exchange (`apikey: {anon-key}`, no JWT):

```
POST /api/v1/auth/oauth/google/token    (or /apple/token)
{"id_token": "<token from the SDK>", "nonce": "<raw nonce, only if you passed one to the SDK>", "name": "<Apple: the name only arrives the first time>"}
```

3. It answers like `/auth/signin`: `{"token","refresh_token","expires_in","provider","user"}`. From there
   everything works the same (REST, storage, push with that JWT).

- Nonce: recommended with Apple. Generate a raw nonce, give the SDK its SHA-256 (Apple) or the value as is
  (Google), and send Rapibase the RAW one; both forms are accepted. If the token carries a nonce and you
  don't send it, or it doesn't match, the answer is 401.
- Linking: the identity (provider + `sub`) stays linked to the user, so Apple keeps working even if it
  hides or changes the email. On the first login it links by verified email to an existing account or
  creates a new one. An email the provider didn't verify is 401, and a token without an email on the first
  login is 400 (request the `email` scope).
- Errors: 401 with the reason (signature, audience, expired, nonce), 404 if the provider isn't configured,
  502 if the provider's keys couldn't be fetched.
- Apple on Android: Apple's Android flow needs a web redirect that isn't supported yet. On Android offer
  Google and email.
- App Store: if the iOS app offers login with Google (or another social provider), Apple requires offering
  Sign in with Apple too.

## Creating the credentials: a guide to walk the person through

The credentials are created by the person (a human) in each provider's console: you have no access. Walk
them through it step by step, one at a time, and confirm each step before the next. Give them their
project's exact callback URL (it is in the instance dashboard → Auth → Providers). At the end they paste
the values in Rapibase Cloud → Settings → Social login, never in the chat.

**Google (web)**, in the Google Cloud console:
1. Top left, create a new project (or pick one) with the app's name.
2. Menu → "Google Auth Platform" (formerly "OAuth consent screen") → "Get started": app name, support
   email, audience "External" and contact email → Create.
3. "Audience": while the app is in "Testing", only the test users you add there can sign in. For anyone to
   sign in, press "Publish app". With the basic scopes (email and profile) Google doesn't ask for
   verification.
4. "Clients" → "Create client" → type "Web application" → under "Authorized redirect URIs" add the
   callback URL (`{base-url}/api/v1/auth/oauth/google/callback`). "Authorized JavaScript origins" can stay
   empty.
5. Copy the "Client ID" and the "Client secret" right then: Google shows the full secret only once (if it
   is lost, create another). They go in `OAUTH_GOOGLE_CLIENT_ID` and `OAUTH_GOOGLE_CLIENT_SECRET`.
- Typical error: "redirect_uri_mismatch" = the URI doesn't match character by character (https,
  subdomain, no trailing slash).

**Google (native app)**, in the same project → Google Auth Platform → Clients:
1. The "Web application" client from the previous step is needed (even if the app has no web): on Android,
   google_sign_in uses it as the serverClientId and the id_token is issued for that ID.
2. Android: "Create client" → type "Android" → package name (applicationId) + SHA-1 fingerprint. The
   development SHA-1 comes from `cd android && ./gradlew signingReport`; for the published app, the one in
   Play Console → Test and release → App integrity → "App signing" (create one Android client per SHA-1).
3. iOS: "Create client" → type "iOS" → bundle ID. Add the "iOS URL scheme" it shows you (the reversed
   client ID) to Info.plist.
4. The Android/iOS client IDs go in `OAUTH_GOOGLE_CLIENT_IDS`, comma-separated. These clients have no
   secret.

**Sign in with Apple**, in the Apple Developer account (requires the paid Apple Developer Program
membership):
1. Certificates, Identifiers & Profiles → Identifiers → the app's App ID → check "Sign in with Apple" →
   Save.
2. In Xcode: target → Signing & Capabilities → "+ Capability" → "Sign in with Apple".
3. `OAUTH_APPLE_CLIENT_IDS` = the bundle ID. No private key or Team ID needed.
4. Only for the web with Apple JS: Identifiers → "+" → "Services IDs" → create one, enable Sign in with
   Apple on it, add the web's domain and return URL, and add that Services ID to `OAUTH_APPLE_CLIENT_IDS`.
- Apple sends the name and the email only on the first login (the user can choose to hide their email:
  an Apple private relay address arrives instead).

**GitHub**, in GitHub → Settings → Developer settings (or the organization → Settings → Developer
settings):
1. "OAuth Apps" → "New OAuth App": name, Homepage URL (the app's web) and "Authorization callback URL" =
   `{base-url}/api/v1/auth/oauth/github/callback` → Register application.
2. Copy the "Client ID"; press "Generate a new client secret" and copy it right away (it is shown once).
   They go in `OAUTH_GITHUB_CLIENT_ID` and `OAUTH_GITHUB_CLIENT_SECRET`.

**Facebook**, in Meta for Developers → My Apps:
1. "Create app" → use case "Authenticate and request data from users with Facebook Login" → name and
   email → Create.
2. Use cases → Facebook Login → Customize → Settings → under "Valid OAuth Redirect URIs" add
   `{base-url}/api/v1/auth/oauth/facebook/callback` → Save. In Permissions, make sure "email" is added.
3. App settings → Basic: copy the "App ID" and the "App secret" (Show button). They go in
   `OAUTH_FACEBOOK_CLIENT_ID` and `OAUTH_FACEBOOK_CLIENT_SECRET`.
4. While the app is in development mode only people with a role in it can sign in. To publish it ("Live"
   mode) Meta asks for a privacy policy URL and a data deletion URL.
