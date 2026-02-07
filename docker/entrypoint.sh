#!/bin/sh
set -e

# Default admin credentials for development
ADMIN_EMAIL="${ADMIN_EMAIL:-admin@localhost}"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-admin}"

# Wait for filesystem to be ready
sleep 1

# Check if admin user exists, create if not
echo "Checking admin user..."
if ! ./micropanel user list 2>/dev/null | grep -q "$ADMIN_EMAIL"; then
    echo "Creating admin user: $ADMIN_EMAIL"
    ./micropanel user create -e "$ADMIN_EMAIL" -p "$ADMIN_PASSWORD" -r admin
else
    echo "Admin user already exists"
fi

echo "Starting MicroPanel..."
exec ./micropanel serve
