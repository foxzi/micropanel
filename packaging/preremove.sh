#!/bin/bash
set -e

# Stop services if running
systemctl stop micropanel 2>/dev/null || true
systemctl stop micropanel-ssl-renew.timer 2>/dev/null || true
systemctl disable micropanel 2>/dev/null || true
systemctl disable micropanel-ssl-renew.timer 2>/dev/null || true

echo "MicroPanel services stopped."
echo "Note: Data in /opt/micropanel/data and /var/www/panel/sites is preserved."
