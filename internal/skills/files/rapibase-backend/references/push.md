# Push notifications — the full contract

NATIVE push to Android (FCM HTTP v1), iOS (APNs) and the web (Web Push/VAPID) with the same API. Every
error is JSON `{"error", "field"?}`: a missing or invalid field answers **400** and `field` says which one.

**Auth of the client endpoints** (`subscribe`, `unsubscribe`, `subscriptions`, `notifications`):
- `apikey: {anon-key}` + `Authorization: Bearer <jwt>`, the user's JWT (the `token` field) → acts on the
  JWT's user (the `user_id` comes from the token; sending another `user_id` gives 403). This is what the
  app uses.
- `apikey: {service-key}` → acts on the `user_id` (UUID from `auth_users`) you pass in the body or in
  `?user_id=`. If the user doesn't exist: 404.
- `POST /api/v1/push/send` is service key ONLY (a user's JWT can never notify others) → from a function,
  `rapibase api` (MCP: `api_post`) or your backend.

## Registering a device — `POST /api/v1/push/subscribe`

| platform | required | what goes in `token` |
|---|---|---|
| `android` | `platform`, `token` | the FCM token: the classic registration token (`getToken()`) **or** the Firebase Installation ID (`register()` → `onRegistered` in firebase_messaging 17 / Android FCM 25.1+). FCM HTTP v1 accepts both in the same `message.token` field; send whichever your SDK version gives you. |
| `ios` | `platform`, `token` | whichever you have: the APNs device token in hex (sent directly through APNs, configuring `/push/config/ios`) **or** the FCM token / Installation ID that Firebase gives on iOS (detected automatically and sent through FCM, which uses the `/push/config/android` config and the APNs key loaded in Firebase). With Flutter + firebase_messaging, send the FCM token with `platform:"ios"`. |
| `web` | `platform`, `endpoint`, `keys.p256dh`, `keys.auth` | optional: by default it is the `endpoint`. Send `{platform:"web", ...subscription.toJSON()}`. |

Optional on all of them: `metadata` (a free object: model, app version…), returned when listing. With the
service key, `user_id`.

```json
POST /api/v1/push/subscribe
{"platform":"android","token":"<fcm token or FID>","metadata":{"app_version":"1.4.0"}}
```

Response: **201** if the device is new and **200** if it was already registered for that user (it updates
`endpoint`/`metadata`). It is idempotent: calling it on every launch and on every token refresh is right.

```json
{"message":"Subscribed successfully","created":true,"reassigned_from":0,
 "subscription":{"id":12,"user_id":"…","platform":"android","token":"…","metadata":{…},"created_at":"…"}}
```

**A token belongs to a single user.** If the same token was already registered for ANOTHER user (for
example, two accounts on the same phone), it moves to the new user and is deleted from the previous one;
`reassigned_from` says how many rows moved. A user can have up to **20 devices**: registering the 21st
drops the one that has gone longest without a refresh (`updated_at`), so abandoned tokens don't pile up.

## Unsubscribing on sign-out — `DELETE /api/v1/push/subscribe`

Body `{"token":"…"}` (on the web, the token is the `endpoint`). If your HTTP client doesn't send a body on
a DELETE, use `?token=…` or `POST /api/v1/push/unsubscribe` with the same body. Call it BEFORE signing
out, because it needs the still-valid JWT. With the service key, `user_id` is optional: without it, it
deletes that token from whoever has it. It is idempotent: `{"message":"Unsubscribed","deleted":1}`; an
unknown token returns `deleted:0` and status 200.

## Listing registered devices (for debugging) — `GET /api/v1/push/subscriptions`

With the service key: `?user_id=<uuid>` lists that user's; without `user_id` it lists everyone's (newest
first, `?limit=` up to 500, 100 by default). With anon key + JWT it returns the user's own. Response:
`{"subscriptions":[{id,user_id,platform,token,endpoint,metadata,created_at}],"count":N}`.
From the agent: `rapibase api get "/api/v1/push/subscriptions?user_id=<uuid>"` (MCP: `api_get`).

## Sending — `POST /api/v1/push/send` (service key)

```jsonc
{
  "user_id": "<uuid>",                 // or "user_ids": [...], or "filter": {...}; none of them = broadcast to everyone
  "title": "Order ready",              // required
  "body": "Come pick it up",
  "data": {"order_id": 123, "screen": "orders"},
  "android": {"channel_id": "orders", "priority": "high", "icon": "ic_stat_notify",
              "color": "#ff6600", "sound": "default", "tag": "order-123",
              "image": "<public image URL>", "click_action": "FLUTTER_NOTIFICATION_CLICK", "ttl_seconds": 3600},
  "ios": {"sound": "default", "badge": 1, "thread_id": "orders", "category": "ORDER"},
  "web": {"icon": "/icon-192.png", "badge": "/badge.png", "image": "…", "url": "/orders/123"},
  "inbox": true                        // true by default; false = push only, it doesn't stay in the inbox
}
```

From the agent: `rapibase api post /api/v1/push/send --data '{"user_id":"<uuid>","title":"Order ready"}'`
(or `--data @notification.json`).

- Target priority: `filter` > `user_ids` (max 1000; for larger audiences use `filter`) > `user_id` >
  broadcast. `filter`: `{role, email_verified, created_after, created_before, metadata:{…}}` over
  `auth_users`. `title` is required, max 255 characters. A `user_id` that doesn't exist gives 404.
- `inbox:false` is for notices that make no sense stored (codes, silent pings, "your order is arriving").
- `data` is a JSON object. **Android:** FCM requires every value to be a string, so strings go as they
  are and the rest (numbers, booleans, objects) is sent serialized as JSON (`123` arrives as `"123"`,
  `{"a":1}` as `"{\"a\":1}"`). **iOS:** the keys go at the root level of the APNs payload, next to `aps`.
  **Web:** it arrives as is in `event.data.json().data`.
- `android`: `channel_id` is the Android 8+ channel (the app creates it; if it doesn't exist Android uses
  the default one); `priority` is `"high"` or `"normal"` (if you omit it, FCM decides); `icon` is the name
  of an app drawable; `sound` is `"default"` or a file in `res/raw`; `tag` replaces the previous
  notification with the same tag. All optional.
- Web: the service worker receives `{title, body, icon, badge, image, url, data}`.

Response **201**:

```json
{"message":"Notification sent","target":"user_id","batch_id":"<uuid>","inbox":1,
 "devices":3,"sent":2,"failed":1,"removed":1,
 "errors":[{"subscription_id":9,"user_id":"…","platform":"android","error":"FCM device token unregistered: …","removed":true}],
 "push_warning":"…"}
```

`inbox` is the inboxes where it was stored, `devices` the devices found, `sent` the ones the provider
accepted and `failed` the rejected ones (`errors` lists up to 50, with the provider's reason). `devices:0`
means that user has nothing registered: the notification stays only in their inbox (unless `inbox` is
false). **Automatic cleanup:** only tokens the provider declares dead are deleted (FCM `404` /
`UNREGISTERED`, APNs `410` / `BadDeviceToken`, Web Push `404` / `410`); they are counted in `removed` and
show up with `removed:true`. Any other error (badly configured credentials, invalid payload, network) does
NOT delete the token: check the `error`.

## In-app inbox — `GET /api/v1/push/notifications`

`?unread=true` returns only the unread ones and `?limit=` accepts up to 100 (50 by default). With the
service key, `?user_id=` is required.

```json
{"notifications":[{"id":45,"user_id":"…","title":"…","body":"…","data":{…},"sent_at":"…","read_at":null,"created_at":"…"}],
 "unread_count":3}
```

To mark one as read: `PUT /api/v1/push/notifications/{id}/read` (404 if it doesn't exist or was already
read). To mark all: `PUT /api/v1/push/notifications/read-all` → `{"count":N}`. Every send stays in each
recipient's inbox, also with `filter` or broadcast (including users with no device, who see it when they
open the app), unless `inbox:false`. Each user marks their own copy as read. In the dashboard's history
each send appears once with its number of recipients.

## Configuration (service key) — the agent leaves it working without the dashboard

With the CLI: `rapibase api get` / `post` / `patch` on these paths (MCP: `api_get` / `api_post` /
`api_update`).

- `POST /api/v1/push/config/web` `{}` generates the VAPID keys and enables web push
  (`rapibase api post /api/v1/push/config/web --data '{}'`). It is IDEMPOTENT: if they already exist it
  returns the public one without regenerating; `{"force":true}` regenerates them and invalidates the
  subscriptions. The web app reads the public key with `GET /api/v1/push/vapid`.
- `POST /api/v1/push/config/ios` `{"key_id","team_id","bundle_id","private_key":"<.p8>","production":true}`
  (the APNs Auth Key comes from Apple Developer → Keys).
- `POST /api/v1/push/config/android` `{"project_id","client_email","private_key"}` (from Firebase's
  service account JSON).
- iOS and Android carry third-party private keys. If the person prefers not to pass them through the
  chat, they load them in the instance dashboard (`rapibase open dashboard`; MCP: `get_project` →
  `dashboard_url`), or save them in a JSON file
  outside the repo and you pass it with `--data @<path>` without opening it.
- `GET /api/v1/push/config` shows the state per platform (no secrets) · `PATCH
  /api/v1/push/config/{platform}/toggle` `{"enabled":bool}` enables or disables it.
- Alternative/complement: a function with `fetch` to the provider API you prefer (FCM directly,
  OneSignal…) with an `FN_` secret. Useful for campaign logic that doesn't fit `/push/send`.
