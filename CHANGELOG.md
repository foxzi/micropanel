# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
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
- License changed to GPL-3.0
