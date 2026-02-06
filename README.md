# MicroPanel

Minimalist static hosting control panel written in Go.

## Features

- Static site hosting management
- Domain binding with SSL (Let's Encrypt)
- ZIP deploy with rollback support
- Redirects configuration
- Basic Auth zones
- File manager
- Audit logging

## Tech Stack

- **Backend:** Go + Gin
- **Frontend:** Templ + HTMX + TailwindCSS
- **Database:** SQLite + golang-migrate
- **Web-server:** Nginx
- **SSL:** Certbot (Let's Encrypt)
- **Containerization:** Docker Compose

## Quick Start

```bash
# Clone repository
git clone https://github.com/yourname/micropanel.git
cd micropanel

# Start with Docker Compose
make dev

# Open browser
open http://localhost:8080
```

Default credentials: `admin@localhost` / `admin`

## Development

```bash
# Run development server
make dev

# Build binary
make build

# Generate templ templates
make templ

# Run migrations
make migrate-up

# Run tests
make test
```

## Configuration

Configuration via environment variables or `config.yaml`:

| Variable | Default | Description |
|----------|---------|-------------|
| `APP_ENV` | `development` | Environment (development/production) |
| `APP_PORT` | `8080` | Application port |
| `APP_SECRET` | - | Session secret key |
| `DB_PATH` | `./data/micropanel.db` | SQLite database path |
| `SITES_PATH` | `/var/www/panel/sites` | Sites root directory |

## Documentation

See [docs/](docs/) for detailed documentation.

## License

GPL-3.0
