#!/bin/bash
# Setup nginx configuration for MicroPanel
set -e

CONFIG_FILE="/etc/micropanel/config.yaml"
TEMPLATE_FILE="/usr/share/micropanel/scripts/panel-nginx.conf.tmpl"
NGINX_CONF="/etc/nginx/sites-enabled/micropanel.conf"

# Extract panel_domain from config
PANEL_DOMAIN=$(grep -E "^\s*panel_domain:" "$CONFIG_FILE" 2>/dev/null | sed 's/.*panel_domain:\s*//' | tr -d '"' | tr -d "'" | xargs)

if [ -z "$PANEL_DOMAIN" ] || [ "$PANEL_DOMAIN" = '""' ] || [ "$PANEL_DOMAIN" = "''" ]; then
    echo "Error: panel_domain is not set in $CONFIG_FILE"
    echo "Please set panel_domain to your panel's domain name."
    exit 1
fi

# Generate nginx config
echo "Generating nginx config for $PANEL_DOMAIN..."
sed "s/PANEL_DOMAIN/$PANEL_DOMAIN/g" "$TEMPLATE_FILE" > "$NGINX_CONF"

# Test and reload nginx
echo "Testing nginx configuration..."
nginx -t

echo "Reloading nginx..."
systemctl reload nginx

echo ""
echo "Done! Panel is now accessible at http://$PANEL_DOMAIN"
echo ""
echo "To enable SSL, run:"
echo "  sudo certbot --nginx -d $PANEL_DOMAIN"
