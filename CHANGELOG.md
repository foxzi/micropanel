# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.2.4] - 2026-02-08

### Added
- certbot and python3-certbot-nginx as package dependencies
- Docker entrypoint script that auto-creates admin user
- Docker-specific config.yaml with development settings

### Changed
- setup-panel-nginx.sh now automatically obtains SSL certificate via certbot
- Use --no-ssl flag to skip SSL certificate generation
- Docker now uses dedicated config instead of local config.yaml
- Updated GitHub Pages with new installation instructions

## [1.2.3] - 2026-02-08

### Changed
- Removed default admin user (admin@localhost)
- Service now requires configuration before starting
- Startup validation checks for panel_domain and admin user
- Clear error messages in logs when configuration is incomplete

### Removed
- Default user creation in migrations

## [1.2.2] - 2026-02-08

### Added
- Panel now listens on 127.0.0.1:8080 by default (localhost only)
- New `panel_domain` config option for nginx setup
- Script to generate nginx reverse proxy config: `/usr/share/micropanel/scripts/setup-panel-nginx.sh`

### Changed
- Added `host` option to config (default: 127.0.0.1)

## [1.2.1] - 2026-02-08

### Fixed
- Enable CGO for SQLite support in release builds

## [1.2.0] - 2026-02-08

### Changed
- Config file now at /etc/micropanel/config.yaml (FHS compliant)
- Database now at /var/lib/micropanel/micropanel.db
- Binary now at /usr/bin/micropanel
- Migrations and static files now at /usr/share/micropanel/
- Removed separate env file, all config in single YAML
- Config file auto-generates random secret on first install

### Fixed
- Logout button now works correctly (CSRF token passed via meta tag to HTMX requests)
- APT repository now correctly separates amd64 and arm64 packages with proper Release file hashes

### Changed
- Server name in dashboard widget is now bold and red

## [1.1.1] - 2026-02-07

### Added
- REST API for site management and deployments (POST/GET/DELETE /api/v1/sites, POST /api/v1/sites/:id/deploy)
- API token authentication via Bearer tokens in Authorization header
- IP whitelist middleware with CIDR notation support
- Security configuration for panel and API access restrictions
- API documentation (docs/ru/api.md, docs/en/api.md)
- Server info widget on dashboard showing external IP, server name, and notes
- Server settings page for configuring server name and notes
- Automatic external IP detection via ifconfig.me on startup
- Settings table in database for storing server configuration

### Fixed
- Generate nginx config when creating site via API
- Project structure initialization
- Go module setup with Gin, Templ, HTMX stack
- SQLite database with golang-migrate migrations
- Configuration system (ENV + YAML support)
- User authentication (login/logout, sessions, bcrypt)
- CSRF middleware protection
- Rate limiting middleware
- Base Templ templates (layout, login, dashboard, site view)
- Docker Compose setup (app + nginx + certbot)
- Makefile with development commands
- Site CRUD operations (create, read, update, delete)
- Site management UI with dashboard
- User model with roles (admin/user)
- Session management with automatic extension
- Installation documentation (RU/EN)
- Domain management (add, delete, set primary)
- Nginx config generation service
- Nginx site config template with SSL support
- Nginx reload with config validation and rollback
- ZIP deploy with secure extraction
- Path traversal and symlink protection
- Atomic directory swap for zero-downtime deploys
- Deploy history with status tracking
- Rollback to previous version
- SSL certificate management via Certbot/Let's Encrypt
- SSL configuration (email, staging mode)
- Certificate issue/renew API endpoints
- SSL status display in site view
- Certificate expiry tracking
- Redirect management (301/302, preserve path/query)
- Redirect CRUD with priority support
- Basic Auth zones with path prefix protection
- Auth zone user management with bcrypt
- htpasswd file generation
- Nginx config integration for redirects and auth
- File manager service with sandbox protection
- File CRUD operations (create, read, write, delete, rename)
- File upload and download
- Text file editing with syntax-aware extensions
- Image preview support
- File manager UI with tree navigation
- Audit logging system with action tracking
- Audit log repository and service
- Audit log UI for admins
- Configurable limits (ZIP size, file size, upload size, sites per user)
- Audit logging integration for all handlers
- Rate limiting for login (5 attempts/min)
- Rate limiting for API (100 requests/min)
- User management (admin): list, create, edit, block/unblock, delete
- User profile page with password change
- SSL auto-renewal scripts (systemd timer/cron)
- SSL status color-coded indicators (valid/expiring/expired)
- GitHub Actions CI/CD for building and releasing packages
- GitHub Pages download page with installation instructions
- CLI commands for user management (list, create, delete, reset-password)
- CLI commands for site management (list, create, delete, enable, disable)
- DEB and RPM package building via nfpm
- Systemd service file for micropanel
- Packaging scripts (preinstall, postinstall, preremove)

### Changed
- Nginx logs moved to /var/log/nginx/{domain}_access.log and {domain}_error.log
- License changed to GPL-3.0
- Simplified domain model: site name is now the primary domain (hostname)
- Domains table renamed to aliases (additional domains for a site)
- SSL certificate is now managed at site level instead of individual domains
- Added www alias toggle (enabled by default) to automatically add www. prefix
- Updated site view UI to show primary domain, www alias checkbox, and alias list
- Updated nginx config generation to use site.GetAllHostnames() for server_name
- Updated CLI site list to show SSL and WWW status
- Added sudo for certbot and nginx commands to run under micropanel user
- Postinstall creates /etc/sudoers.d/micropanel with NOPASSWD rules
- Added ZIP archive requirements hint in deploy section UI
