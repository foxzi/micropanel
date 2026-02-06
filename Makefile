.PHONY: dev build templ migrate-up migrate-down test clean

# Development
dev:
	docker compose up --build

dev-down:
	docker compose down

dev-logs:
	docker compose logs -f app

# Build
build:
	go build -o bin/micropanel ./cmd/micropanel

build-docker:
	docker compose build

# Templ
templ:
	templ generate

templ-watch:
	templ generate --watch

# Migrations
migrate-up:
	docker compose exec app ./micropanel migrate up

migrate-down:
	docker compose exec app ./micropanel migrate down

# Test
test:
	go test -v ./...

test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Clean
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html
	docker compose down -v

# Local development (without docker)
local-run: templ
	go run ./cmd/micropanel

local-db:
	mkdir -p data
	sqlite3 data/micropanel.db < migrations/001_init.up.sql

# Help
help:
	@echo "Available targets:"
	@echo "  dev          - Start development environment with Docker Compose"
	@echo "  dev-down     - Stop development environment"
	@echo "  dev-logs     - Show application logs"
	@echo "  build        - Build binary"
	@echo "  templ        - Generate templ templates"
	@echo "  templ-watch  - Watch and regenerate templ templates"
	@echo "  migrate-up   - Run database migrations"
	@echo "  migrate-down - Rollback last migration"
	@echo "  test         - Run tests"
	@echo "  clean        - Clean build artifacts"
	@echo "  local-run    - Run locally without Docker"
	@echo "  help         - Show this help"
