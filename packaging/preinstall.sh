#!/bin/bash
set -e

# Create micropanel user if not exists
if ! id -u micropanel >/dev/null 2>&1; then
    useradd --system --no-create-home --shell /sbin/nologin micropanel
fi

# Add micropanel to nginx group for config access
usermod -aG nginx micropanel 2>/dev/null || usermod -aG www-data micropanel 2>/dev/null || true
