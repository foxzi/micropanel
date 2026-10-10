# REST API

MicroPanel provides a REST API for automating site management and deployments.

## Enabling the API

Add to `config.yaml`:

```yaml
api:
  enabled: true
  tokens: []  # Tokens can be created via web panel
```

Or via environment variable:

```bash
API_ENABLED=true
```

## Token Management

### Via Web Panel (Recommended)

All users can create and manage their API tokens through the web panel:
1. Log in to the panel
2. Navigate to "API Tokens" in the navigation bar
3. Click "Create Token" and enter a name
4. Copy the token (it's only shown once!)

Tokens are associated with the user and only grant access to their sites.

### Via Config File (for backward compatibility)

You can also add tokens in `config.yaml`:

```yaml
api:
  enabled: true
  tokens:
    - name: "deploy-bot"
      token: "your-secret-token-here"
      user_id: 1  # Owner user ID
    - name: "ci-cd"
      token: "another-secret-token"
      user_id: 2
```

> **Important:** The `user_id` field is required. Tokens without `user_id` will be rejected.

## Authentication

The API uses Bearer tokens. Add the `Authorization` header to each request:

```
Authorization: Bearer your-secret-token-here
```

## IP Access Restriction

You can restrict API access by IP addresses. CIDR notation is supported:

```yaml
security:
  # Whitelist for web panel (empty list = allow all)
  panel_allowed_ips:
    - "192.168.1.0/24"
    - "10.0.0.1"

  # Whitelist for API (empty list = allow all)
  api_allowed_ips:
    - "192.168.1.100"
    - "10.0.0.0/8"
```

## Endpoints

Every route requires a token with a `user_id`. A token sees only its user's
sites; an administrator's token sees all sites. Another user's site answers
`403 Forbidden`.

### Site object

Routes that return a site answer with the same object:

```json
{
  "id": 1,
  "name": "example.com",
  "is_enabled": true,
  "ssl_enabled": true,
  "fix_mime_types": false,
  "ssl_status": "issued",
  "ssl_expires_at": "2026-12-31T10:00:00Z"
}
```

| Field | Description |
|-------|-------------|
| `ssl_enabled` | the site has a certificate and nginx serves it over HTTPS |
| `ssl_status` | state of the latest certificate issuance: `none` (never issued), `issuing` (running), `issued` (succeeded), `failed` |
| `ssl_error` | why the latest issuance failed (only with `failed`; the tail of the certbot output, up to 1000 characters) |
| `ssl_expires_at` | certificate expiry (RFC 3339), if there is a certificate |

An issuance interrupted by a panel restart becomes `failed` at startup with
the reason `certificate issuance interrupted by a panel restart`.

### Create Site

```
POST /api/v1/sites
```

**Request body:**
```json
{
  "name": "example.com",
  "ssl": false,
  "fix_mime_types": false
}
```

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| name | string | yes | - | Site domain name |
| ssl | bool | no | false | Issue a certificate right after creation (synchronously); usually the certificate is requested separately with `POST /api/v1/sites/:id/ssl` |
| fix_mime_types | bool | no | false | Fix MIME types of files with an encoded query string in the name |

The site is live at once: nginx serves a placeholder until an archive is
deployed.

**Response:** `201 Created` with the site object. If a site with this name
already exists, it is returned with `200 OK`: creation is idempotent by name.

> **Important:** on a name match the existing site is returned even if it
> belongs to another user. Check access with `GET /api/v1/sites/:id`: another
> user's site answers `403`.

**Errors:**
- `400 Bad Request` — name is missing
- `401 Unauthorized` — invalid token
- `403 Forbidden` — the token has no `user_id`
- `500 Internal Server Error` — the site was not created or the nginx configuration failed

### List Sites

```
GET /api/v1/sites
```

**Response (200 OK):** an array of site objects.

### Get Site

```
GET /api/v1/sites/:id
```

**Response (200 OK):** the site object.

**Errors:**
- `400 Bad Request` — invalid ID
- `403 Forbidden` — another user's site
- `404 Not Found` — site not found

### Delete Site

```
DELETE /api/v1/sites/:id
```

Deletes the certificate (if any), the nginx configuration and the site.

**Response (200 OK):**
```json
{
  "message": "site deleted"
}
```

**Errors:**
- `400 Bad Request` — invalid ID
- `403 Forbidden` — another user's site
- `404 Not Found` — site not found

### Deploy Archive

```
POST /api/v1/sites/:id/deploy
```

**Content-Type:** `multipart/form-data`

**Parameters:**
- `file` — the archive (`.zip`, `.tar.gz` or `.tgz`). A root directory shared
  by all files of the archive is stripped.

**Headers:**
- `Idempotency-Key` (optional) — a key for safe retries: 1–255 visible ASCII
  characters, unique per site.

The archive is unpacked next to the current version and replaces it with two
renames; the previous version stays in `public_prev`. Deploys of one site run
one at a time.

**Response (200 OK):**
```json
{
  "deploy_id": 1,
  "status": "success"
}
```

**Retrying with `Idempotency-Key`:**

| Deploy with this key | Response |
|----------------------|----------|
| none | a normal deploy |
| succeeded | `200 OK` with the earlier `deploy_id` and `"replayed": true`; the archive is not unpacked and `public_prev` is untouched |
| still running | `409 Conflict` with the running `deploy_id` |
| failed | a new attempt in the same record (same `deploy_id`) |

Without the header every request is a new deploy. A deploy interrupted by a
panel restart becomes `failed` at startup, and its key can be retried.

**Errors:**
- `400 Bad Request` — no file, invalid `Idempotency-Key`, unsupported archive, too many files, path traversal, symbolic links
- `403 Forbidden` — another user's site
- `404 Not Found` — site not found
- `409 Conflict` — a deploy with this `Idempotency-Key` is still running
- `413 Request Entity Too Large` — archive over 100 MB

Archive limits are built in: archive up to 100 MB, file up to 10 MB, up to
10,000 files, path up to 500 characters, any path with `..` is rejected.

### List Deploys

```
GET /api/v1/sites/:id/deploys
```

**Query parameters:**

| Parameter | Default | Description |
|-----------|---------|-------------|
| limit | 20 | how many latest deploys to return, at most 100 |
| idempotency_key | - | return only the deploy with this key (an empty array if there is none) |

**Response (200 OK):** deploys, newest first.
```json
[
  {
    "id": 2,
    "status": "failed",
    "idempotency_key": "publication-7",
    "error": "extract archive: ...",
    "created_at": "2026-10-10T12:00:00Z"
  }
]
```

`status` is `pending`, `success` or `failed`.

**Errors:**
- `400 Bad Request` — invalid ID or limit
- `403 Forbidden` — another user's site
- `404 Not Found` — site not found

### Issue Certificate

```
POST /api/v1/sites/:id/ssl
```

A Let's Encrypt certificate through certbot (`--webroot`). The site's names
must already point at the panel server.

**Request body (optional):**
```json
{
  "mode": "all",
  "async": true
}
```

| Parameter | Default | Description |
|-----------|---------|-------------|
| mode | `all` | `all` — the site name, `www.` (if enabled) and aliases; `primary` — the site name and `www.`; `aliases` — aliases only; `none` — issue nothing |
| async | false | issue in the background |

**Synchronously** (no `async`): the response comes when certbot finishes (up
to 5 minutes) — `200 OK` with the site object, or `500` on failure.

**Asynchronously** (`"async": true`): `202 Accepted` at once with the site
object in `ssl_status: "issuing"`; read the outcome from
`GET /api/v1/sites/:id` (`issued`, or `failed` with `ssl_error`). A repeated
request while the issuance runs is also `202` and does not start a second
certbot. Issuances of all sites on the panel run one at a time.

**Errors:**
- `400 Bad Request` — invalid ID, invalid mode, no names to issue for
- `403 Forbidden` — another user's site
- `404 Not Found` — site not found
- `500 Internal Server Error` — a synchronous issuance failed

### Site Aliases

Additional names served by the same site.

```
GET /api/v1/sites/:id/domains
POST /api/v1/sites/:id/domains
DELETE /api/v1/sites/:id/domains/:domainId
```

`POST` takes `{"hostname": "www.example.org"}` and answers `201 Created`:

```json
{
  "id": 3,
  "site_id": 1,
  "hostname": "www.example.org"
}
```

Adding the same name to the same site again answers `200 OK` with the
existing alias; a name taken by another site is `409 Conflict`. The site name
and its `www.` cannot be aliases (`400`). `GET` returns an array of aliases,
`DELETE` returns `{"message": "domain alias deleted"}`. To cover aliases with
a certificate use `POST /api/v1/sites/:id/ssl` with `mode` `all` or
`aliases`.

## Usage Examples

### cURL

```bash
# Create a site (no certificate by default)
curl -X POST http://localhost:8080/api/v1/sites \
  -H "Authorization: Bearer your-secret-token" \
  -H "Content-Type: application/json" \
  -d '{"name": "example.com"}'

# Request a certificate in the background and read the outcome
curl -X POST http://localhost:8080/api/v1/sites/1/ssl \
  -H "Authorization: Bearer your-secret-token" \
  -H "Content-Type: application/json" \
  -d '{"mode": "primary", "async": true}'
curl http://localhost:8080/api/v1/sites/1 \
  -H "Authorization: Bearer your-secret-token"

# List sites
curl http://localhost:8080/api/v1/sites \
  -H "Authorization: Bearer your-secret-token"

# Get site info
curl http://localhost:8080/api/v1/sites/1 \
  -H "Authorization: Bearer your-secret-token"

# Deploy an archive (a retry with the same key does not unpack it again)
curl -X POST http://localhost:8080/api/v1/sites/1/deploy \
  -H "Authorization: Bearer your-secret-token" \
  -H "Idempotency-Key: release-42" \
  -F "file=@site.zip"

# Latest deploys
curl "http://localhost:8080/api/v1/sites/1/deploys?limit=5" \
  -H "Authorization: Bearer your-secret-token"

# Delete site
curl -X DELETE http://localhost:8080/api/v1/sites/1 \
  -H "Authorization: Bearer your-secret-token"
```

### GitHub Actions

```yaml
name: Deploy

on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Create archive
        run: zip -r site.zip . -x ".git/*"

      - name: Deploy to MicroPanel
        run: |
          curl -X POST ${{ secrets.MICROPANEL_URL }}/api/v1/sites/${{ secrets.SITE_ID }}/deploy \
            -H "Authorization: Bearer ${{ secrets.MICROPANEL_TOKEN }}" \
            -F "file=@site.zip"
```

### Python

```python
import requests

API_URL = "http://localhost:8080/api/v1"
TOKEN = "your-secret-token"

headers = {"Authorization": f"Bearer {TOKEN}"}

# Create site
response = requests.post(
    f"{API_URL}/sites",
    headers=headers,
    json={"name": "example.com"}
)
site = response.json()
print(f"Created site: {site['id']}")

# Deploy
with open("site.zip", "rb") as f:
    response = requests.post(
        f"{API_URL}/sites/{site['id']}/deploy",
        headers=headers,
        files={"file": f}
    )
print(response.json())
```

## Rate Limiting

The API is limited to 100 requests per minute per IP address.

## Response Codes

| Code | Description |
|------|-------------|
| 200 | Successful request |
| 201 | Resource created |
| 202 | Accepted, running in the background |
| 400 | Bad request |
| 401 | Unauthorized |
| 403 | Forbidden (IP not in whitelist, another user's site, token without user_id) |
| 404 | Resource not found |
| 409 | Conflict (alias taken by another site, deploy with this key still running) |
| 413 | Request entity too large |
| 429 | Too many requests |
| 500 | Internal server error |
