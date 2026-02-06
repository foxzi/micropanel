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

## Installation

### From packages (recommended)

Download packages from [Releases](https://github.com/foxzi/micropanel/releases) or [Downloads page](https://foxzi.github.io/micropanel/).

**Debian/Ubuntu:**
```bash
sudo dpkg -i micropanel_1.0.0-1_amd64.deb
sudo apt-get install -f  # install dependencies
```

**RHEL/Fedora:**
```bash
sudo rpm -i micropanel-1.0.0-1.x86_64.rpm
```

**Start service:**
```bash
# Edit configuration
sudo nano /etc/micropanel/micropanel.env

# Start and enable service
sudo systemctl enable --now micropanel

# Check status
sudo systemctl status micropanel
```

Default credentials: `admin@localhost` / `admin`

### From source (Docker)

```bash
git clone https://github.com/foxzi/micropanel.git
cd micropanel
make dev
```

Open http://localhost:8080

### CLI commands

```bash
# Start web server
micropanel serve

# User management
micropanel user list
micropanel user create -e admin@example.com -p password -r admin
micropanel user reset-password admin@example.com -p newpassword

# Site management
micropanel site list
micropanel site create -n "my-site" -o 1
micropanel site enable 1
```

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
