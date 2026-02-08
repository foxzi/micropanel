# REST API

MicroPanel provides a REST API for automating site management and deployments.

## Enabling the API

Add to `config.yaml`:

```yaml
api:
  enabled: true
  tokens:
    - name: "deploy-bot"
      token: "your-secret-token-here"
    - name: "ci-cd"
      token: "another-secret-token"
```

Or via environment variable:

```bash
API_ENABLED=true
```

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

### Create Site

```
POST /api/v1/sites
```

**Request body:**
```json
{
  "name": "example.com",
  "ssl": true
}
```

**Parameters:**
| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| name | string | yes | - | Site domain name |
| ssl | bool | no | true | Automatically issue SSL certificate |

**Response (201 Created):**
```json
{
  "id": 1,
  "name": "example.com",
  "is_enabled": true,
  "ssl_enabled": false
}
```

> **Note:** `ssl_enabled` in response shows current status. Certificate is issued asynchronously, so it will be `false` right after creation. Status will update after successful certificate issuance.

**Errors:**
- `400 Bad Request` - name not provided
- `401 Unauthorized` - invalid token
- `409 Conflict` - site with this name already exists

### List Sites

```
GET /api/v1/sites
```

**Response (200 OK):**
```json
[
  {
    "id": 1,
    "name": "example.com",
    "is_enabled": true,
    "ssl_enabled": true
  },
  {
    "id": 2,
    "name": "test.com",
    "is_enabled": true,
    "ssl_enabled": false
  }
]
```

### Get Site

```
GET /api/v1/sites/:id
```

**Response (200 OK):**
```json
{
  "id": 1,
  "name": "example.com",
  "is_enabled": true,
  "ssl_enabled": true
}
```

**Errors:**
- `400 Bad Request` - invalid ID
- `404 Not Found` - site not found

### Delete Site

```
DELETE /api/v1/sites/:id
```

**Response (200 OK):**
```json
{
  "message": "site deleted"
}
```

**Errors:**
- `400 Bad Request` - invalid ID
- `404 Not Found` - site not found

### Deploy Archive

```
POST /api/v1/sites/:id/deploy
```

**Content-Type:** `multipart/form-data`

**Parameters:**
- `file` - archive file (ZIP or TAR.GZ)

**Response (200 OK):**
```json
{
  "deploy_id": 1,
  "status": "success"
}
```

**Errors:**
- `400 Bad Request` - file not provided or invalid format
- `404 Not Found` - site not found
- `413 Request Entity Too Large` - archive too large (max 100MB)

## Usage Examples

### cURL

```bash
# Create site with SSL (default)
curl -X POST http://localhost:8080/api/v1/sites \
  -H "Authorization: Bearer your-secret-token" \
  -H "Content-Type: application/json" \
  -d '{"name": "example.com"}'

# Create site without SSL
curl -X POST http://localhost:8080/api/v1/sites \
  -H "Authorization: Bearer your-secret-token" \
  -H "Content-Type: application/json" \
  -d '{"name": "example.com", "ssl": false}'

# List sites
curl http://localhost:8080/api/v1/sites \
  -H "Authorization: Bearer your-secret-token"

# Get site info
curl http://localhost:8080/api/v1/sites/1 \
  -H "Authorization: Bearer your-secret-token"

# Deploy archive
curl -X POST http://localhost:8080/api/v1/sites/1/deploy \
  -H "Authorization: Bearer your-secret-token" \
  -F "file=@site.zip"

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
| 400 | Bad request |
| 401 | Unauthorized |
| 403 | Forbidden (IP not in whitelist) |
| 404 | Resource not found |
| 409 | Conflict (resource already exists) |
| 413 | Request entity too large |
| 429 | Too many requests |
| 500 | Internal server error |
