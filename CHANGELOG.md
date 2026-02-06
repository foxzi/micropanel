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
