# Storage — /api/v1/storage/*

Files in buckets. The rules (public or private bucket, limits, owner of each file) live in the
instance's Postgres, and the instance itself serves the files: there is never an S3 URL in between.

- **Public bucket**: anyone reads its files at `{base-url}/api/v1/storage/public/{bucket}/{path}`, with no
  keys (what goes in an `<img src>`). Uploading, replacing and deleting still require permission.
- **Private bucket** (the default): read with the SERVICE_KEY, with ANON_KEY + JWT of the user who **owns**
  the file, or with a **signed URL** (temporary, no keys).
- **Owner**: what a user uploads with ANON_KEY + JWT is theirs; only they can read it (if the bucket is
  private), replace it or delete it. What is uploaded with the SERVICE_KEY has no owner, unless you pass
  `owner_id`.
- **Per-bucket limits**: `file_size_limit` (bytes) and `allowed_mime_types` (`["image/*","application/pdf"]`).
  An upload that doesn't comply answers 413 or 415 with the reason. `file_size_limit: 0` and
  `allowed_mime_types: []` mean no limit.
- **Bucket names**: 3–63 characters, lowercase letters, digits, dots and hyphens, starting and ending with
  a letter or digit. `buckets`, `public`, `sign`, `upload`, `owner`, `object`, `objects` and `rules` are
  reserved.
- **Managing buckets and access rules takes the SERVICE_KEY** (list, create, change, delete): from the
  agent with `rapibase api` or the MCP tools, never from the app's frontend.
- Downloads support `Range` (video and audio can be seeked), `ETag`/304 and caching. HTML and SVG are
  served isolated (CSP sandbox): an uploaded file never runs scripts on the instance's domain.

MCP: `create_bucket` / `update_bucket` / `delete_bucket` / `list_buckets`, `upload_object` /
`list_objects` / `download_object` (up to 10 MB) / `delete_object` (a path ending in `/` deletes the whole
folder) and `create_signed_url`. With the CLI (example names; `-F` uploads a file as multipart, `-o` saves
a download to a file):

```bash
# create a bucket: public, max 5 MB per file, images only (MCP: create_bucket)
rapibase api post /api/v1/storage/buckets --data '{"name":"my-bucket","public":true,"file_size_limit":5242880,"allowed_mime_types":["image/*"]}'
# change settings (only what you send) / delete (force=true also deletes the files) (MCP: update_bucket / delete_bucket)
rapibase api patch /api/v1/storage/buckets/my-bucket --data '{"public":false}'
rapibase api delete "/api/v1/storage/buckets/my-bucket?force=true"

# upload a local file (multipart: path = folder, or key = full path; it replaces a file at the same path
# unless -F upsert=false) / download one to a file (MCP: upload_object / download_object)
rapibase api post /api/v1/storage/my-bucket/upload -F file=@./photo.png -F path=folder/
rapibase api get /api/v1/storage/my-bucket/folder/photo.png -o photo.png

# list ONE folder level: files, and sub-folders with is_dir (MCP: list_objects) / delete one (MCP: delete_object)
rapibase api get "/api/v1/storage/my-bucket/list?prefix=folder/"
rapibase api delete /api/v1/storage/my-bucket/folder/photo.png
# delete several / move / copy (MCP: api_post)
rapibase api post /api/v1/storage/my-bucket/remove --data '{"paths":["a.png","b.png"]}'
rapibase api post /api/v1/storage/my-bucket/move --data '{"from":"a.png","to":"old/a.png"}'   # or /copy

# signed URL to read without keys (expires_in in seconds, max 7 days; "download":true forces the download)
# (MCP: create_signed_url)
rapibase api post /api/v1/storage/sign/my-bucket/folder/photo.png --data '{"expires_in":3600}'
# several at once (MCP: api_post): POST /api/v1/storage/sign/my-bucket with {"paths":[...],"expires_in":3600}
```

Uploading and reading from the app's code:

```js
// upload (multipart): path = folder, or key = full path. upsert is true by default (replaces a file at
// the same path); upsert=false answers 409 if it already exists;
// cache_control = seconds of browser cache for public buckets (default 3600)
const form = new FormData()
form.append('file', file)
form.append('path', 'folder/')
form.append('metadata', JSON.stringify({ k: 'v' }))
const up = await fetch(`${RAPIBASE_URL}/api/v1/storage/my-bucket/upload`, {
  method: 'POST',
  headers: { apikey: RAPIBASE_ANON_KEY, Authorization: 'Bearer ' + jwt },
  body: form,
})
// the response carries url (needs keys) and public_url if the bucket is public

// read: GET /api/v1/storage/my-bucket/folder/photo.png with the same headers
```

Direct upload from the browser without passing the file through your backend: the logged-in user asks for
a signed upload URL (valid 2 hours, and the file is theirs) and does a PUT with the file:

```js
const r = await fetch(RAPIBASE_URL + '/api/v1/storage/upload/sign/my-bucket/' + userId + '/avatar.png', {
  method: 'POST',
  headers: { apikey: RAPIBASE_ANON_KEY, Authorization: 'Bearer ' + jwt, 'Content-Type': 'application/json' },
  body: JSON.stringify({ upsert: true }),
})
const { signed_url } = await r.json()
await fetch(signed_url, { method: 'PUT', headers: { 'Content-Type': file.type }, body: file })
```

Also: `/search?key=...&value=...`, `/metadata/*` (GET/PATCH), `/api/v1/storage/owner/:owner_id` (from the
agent: `rapibase api`; MCP: `api_get` / `api_update`). With ANON_KEY + JWT, listing or searching returns
only what the rules let the user read.

## Access rules (teams, sharing, whatever the app needs)

Beyond "owner or public" you add **SQL rules per bucket**: Postgres applies them as RLS on the files,
evaluated as the user (ANON_KEY + JWT). The SERVICE_KEY is never filtered. A rule is a condition on the
file's row: columns `bucket_id`, `name` (full path), `owner`, `size`, `mime_type`, `metadata`; functions
`auth.uid()`, `auth.email()`, `auth.jwt()`, `storage.foldername(name)` (folders as text[]),
`storage.filename(name)`, `storage.extension(name)`; and any table the user can read (its own RLS
applies).

- Base that already exists: public buckets readable by everyone and, with `owner_access` on (the default),
  each user uploads files that are theirs and reads/replaces/deletes their own.
- **Permissive** rule (the default): adds access. **Restrictive** rule (`"restrictive": true`): an extra
  requirement every access must meet. `owner_access: false` on the bucket = only your rules count.
- `operation`: `read`, `upload`, `update` (replace the file or its metadata), `delete` or `all`.
- MCP: `set_storage_rule` / `list_storage_rules` / `delete_storage_rule`. Invalid SQL comes back with the
  Postgres error.
- The listing also shows the built-in rules (`rapibase_storage_*`, marked `builtin: true`): they are the
  base above and can't be deleted; to drop the owner model on a bucket, set `owner_access: false`. Your
  own rule names use lowercase letters, digits and `_`, and can't start with `rapibase_`.

```bash
# folders per team ("{team_id}/file"): members read and upload, only whoever uploaded deletes
rapibase api patch /api/v1/storage/buckets/docs --data '{"owner_access":false}'
rapibase api post /api/v1/storage/rules --data '{
  "name":"team_read","bucket":"docs","operation":"read",
  "condition":"EXISTS (SELECT 1 FROM team_members m WHERE m.user_id = auth.uid() AND m.team_id::text = (storage.foldername(name))[1])"}'
rapibase api post /api/v1/storage/rules --data '{
  "name":"team_upload","bucket":"docs","operation":"upload",
  "condition":"owner = auth.uid() AND EXISTS (SELECT 1 FROM team_members m WHERE m.user_id = auth.uid() AND m.team_id::text = (storage.foldername(name))[1])"}'
rapibase api post /api/v1/storage/rules --data '{
  "name":"team_delete","bucket":"docs","operation":"delete","condition":"owner = auth.uid()"}'

# on top of the owner model: each user can only upload inside their folder "{their id}/"
rapibase api post /api/v1/storage/rules --data '{
  "name":"own_folder","bucket":"avatars","operation":"upload","restrictive":true,
  "condition":"(storage.foldername(name))[1] = auth.uid()::text"}'

rapibase api get "/api/v1/storage/rules?bucket=docs"
rapibase api delete /api/v1/storage/rules/team_delete
```

If the rule queries a table, that table has to be readable by the user (e.g. `team_members` with RLS
`owner` on `user_id`: each one sees their memberships), or use a `SECURITY DEFINER` function created with
`rapibase sql` (MCP: `execute_sql`).

To check a rule as a user, repeat the call with a signed-in test user's JWT instead of the service key:
`rapibase api get "/api/v1/storage/docs/list?prefix=<team_id>/" --jwt <token>` ([credentials.md](credentials.md)).
If storage itself fails (every upload gives 5xx), read its logs: `rapibase logs --container storage` (MCP:
`get_logs` with `container`).
