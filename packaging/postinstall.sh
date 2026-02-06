#!/bin/bash
set -e

# Set ownership
chown -R micropanel:micropanel /opt/micropanel/data
chown -R micropanel:micropanel /var/www/panel/sites

# Create config from example if not exists
if [ ! -f /etc/micropanel/micropanel.env ]; then
    cat > /etc/micropanel/micropanel.env <<EOF
# MicroPanel configuration
# See /opt/micropanel/config.yaml.example for all options

APP_ENV=production
APP_SECRET=$(openssl rand -hex 32)
DB_PATH=/opt/micropanel/data/micropanel.db
SITES_PATH=/var/www/panel/sites
NGINX_CONFIG_PATH=/etc/nginx/sites-enabled
EOF
    chmod 600 /etc/micropanel/micropanel.env
fi

# Reload systemd
systemctl daemon-reload

echo ""
echo "MicroPanel installed successfully!"
echo ""
echo "Next steps:"
echo "  1. Edit /etc/micropanel/micropanel.env"
echo "  2. Start service: systemctl start micropanel"
echo "  3. Enable on boot: systemctl enable micropanel"
echo "  4. Access panel at http://localhost:8080"
echo ""
