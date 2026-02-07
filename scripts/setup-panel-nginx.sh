#!/bin/bash
# Setup nginx configuration for MicroPanel with SSL
set -e

CONFIG_FILE="/etc/micropanel/config.yaml"
TEMPLATE_FILE="/usr/share/micropanel/scripts/panel-nginx.conf.tmpl"
NGINX_CONF="/etc/nginx/sites-enabled/micropanel.conf"

# Parse arguments
SKIP_SSL=false
for arg in "$@"; do
    case $arg in
        --no-ssl)
            SKIP_SSL=true
            ;;
    esac
done

# Extract panel_domain from config
PANEL_DOMAIN=$(grep -E "^\s*panel_domain:" "$CONFIG_FILE" 2>/dev/null | sed 's/.*panel_domain:\s*//' | tr -d '"' | tr -d "'" | xargs)

if [ -z "$PANEL_DOMAIN" ] || [ "$PANEL_DOMAIN" = '""' ] || [ "$PANEL_DOMAIN" = "''" ]; then
    echo "Error: panel_domain is not set in $CONFIG_FILE"
    echo "Please set panel_domain to your panel's domain name."
    exit 1
fi

# Extract ssl.email from config (optional)
SSL_EMAIL=$(grep -E "^\s*email:" "$CONFIG_FILE" 2>/dev/null | head -1 | sed 's/.*email:\s*//' | tr -d '"' | tr -d "'" | xargs)

# Generate nginx config
echo "Generating nginx config for $PANEL_DOMAIN..."
sed "s/PANEL_DOMAIN/$PANEL_DOMAIN/g" "$TEMPLATE_FILE" > "$NGINX_CONF"

# Test and reload nginx
echo "Testing nginx configuration..."
nginx -t

echo "Reloading nginx..."
systemctl reload nginx

echo ""
echo "Nginx configured! Panel is now accessible at http://$PANEL_DOMAIN"

# SSL certificate
if [ "$SKIP_SSL" = true ]; then
    echo ""
    echo "SSL skipped. To enable SSL later, run:"
    echo "  sudo certbot --nginx -d $PANEL_DOMAIN"
    exit 0
fi

# Check if certbot is installed
if ! command -v certbot &> /dev/null; then
    echo ""
    echo "Certbot not found. To enable SSL, install certbot and run:"
    echo "  sudo apt install certbot python3-certbot-nginx"
    echo "  sudo certbot --nginx -d $PANEL_DOMAIN"
    exit 0
fi

echo ""
echo "Obtaining SSL certificate..."

if [ -n "$SSL_EMAIL" ]; then
    certbot --nginx -d "$PANEL_DOMAIN" --non-interactive --agree-tos -m "$SSL_EMAIL"
else
    certbot --nginx -d "$PANEL_DOMAIN" --non-interactive --agree-tos --register-unsafely-without-email
fi

echo ""
echo "Done! Panel is now accessible at https://$PANEL_DOMAIN"
