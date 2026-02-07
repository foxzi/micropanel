# MicroPanel Installation

## Requirements

- Docker and Docker Compose
- Git

## Quick Start

```bash
# Clone repository
git clone https://github.com/foxzi/micropanel.git
cd micropanel

# Start with Docker Compose
make dev

# Open in browser
open http://localhost:8081
```

## Default Credentials

- **Email:** admin@localhost
- **Password:** admin

**Important:** Change the password after first login!

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `APP_ENV` | `development` | Environment (development/production) |
| `APP_PORT` | `8080` | Application port |
| `APP_SECRET` | - | Session secret key |
| `DB_PATH` | `./data/micropanel.db` | SQLite database path |
| `SITES_PATH` | `/var/www/panel/sites` | Sites root directory |

### Configuration File

Copy `config.yaml.example` to `config.yaml` and configure:

```yaml
app:
  env: production
  port: 8080
  secret: your-secret-key

database:
  path: /app/data/micropanel.db

sites:
  path: /var/www/panel/sites

nginx:
  config_path: /etc/nginx/sites-enabled
  reload_cmd: nginx -s reload
```

## Directory Structure

```
micropanel/
├── cmd/micropanel/     # Entry point
├── internal/           # Internal packages
│   ├── config/         # Configuration
│   ├── database/       # Database
│   ├── handlers/       # HTTP handlers
│   ├── middleware/     # Middleware
│   ├── models/         # Data models
│   ├── repository/     # Repositories
│   ├── services/       # Business logic
│   └── templates/      # Templ templates
├── migrations/         # SQL migrations
├── web/static/         # Static files
├── docker/             # Docker files
└── docs/               # Documentation
```
