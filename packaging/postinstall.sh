#!/bin/bash
set -e

# Set ownership
chown -R micropanel:micropanel /var/lib/micropanel
chown -R micropanel:micropanel /var/www/panel/sites
chown root:micropanel /etc/micropanel/config.yaml
chmod 640 /etc/micropanel/config.yaml

# Create certbot webroot for ACME challenges
mkdir -p /var/www/certbot
chown root:root /var/www/certbot
chmod 755 /var/www/certbot

# Generate self-signed certificate for nginx default_server (unknown domains)
SSL_DIR="/etc/micropanel/ssl"
SSL_CRT="$SSL_DIR/default.crt"
SSL_KEY="$SSL_DIR/default.key"
if [ ! -f "$SSL_CRT" ] || [ ! -f "$SSL_KEY" ]; then
    mkdir -p "$SSL_DIR"
    chmod 750 "$SSL_DIR"
    openssl req -x509 -nodes -newkey rsa:2048 -days 3650 \
        -keyout "$SSL_KEY" -out "$SSL_CRT" \
        -subj "/CN=micropanel-default" >/dev/null 2>&1
    chmod 600 "$SSL_KEY"
    chmod 644 "$SSL_CRT"
fi

# Create sudoers file for micropanel user
cat > /etc/sudoers.d/micropanel <<EOF
# Allow micropanel to run certbot and nginx without password
micropanel ALL=(ALL) NOPASSWD: /usr/bin/certbot
micropanel ALL=(ALL) NOPASSWD: /usr/sbin/nginx
micropanel ALL=(ALL) NOPASSWD: /usr/bin/systemctl restart nginx
micropanel ALL=(ALL) NOPASSWD: /usr/bin/tee /etc/nginx/sites-enabled/*
micropanel ALL=(ALL) NOPASSWD: /usr/bin/rm -f /etc/nginx/sites-enabled/*
micropanel ALL=(ALL) NOPASSWD: /usr/bin/cat /etc/letsencrypt/live/*/fullchain.pem
EOF
chmod 440 /etc/sudoers.d/micropanel

# Generate random secret if default is present
if grep -q "change-me-min-32-chars" /etc/micropanel/config.yaml 2>/dev/null; then
    SECRET=$(openssl rand -hex 32)
    sed -i "s/change-me-min-32-chars-random-string/$SECRET/" /etc/micropanel/config.yaml
fi

# Restart nginx if running (picks up /etc/nginx/conf.d/micropanel.conf)
# Deferred until after systemd limits / nginx.conf tuning below.

# Raise nginx file descriptor limit (each hosted site opens access+error logs)
mkdir -p /etc/systemd/system/nginx.service.d
if [ ! -f /etc/systemd/system/nginx.service.d/micropanel-limits.conf ]; then
    cat > /etc/systemd/system/nginx.service.d/micropanel-limits.conf <<'EOF'
# Installed by micropanel - raises NOFILE so nginx can serve many vhosts
[Service]
LimitNOFILE=65536
EOF
fi

# Bump nginx worker limits if still at packaged defaults
if [ -f /etc/nginx/nginx.conf ]; then
    if ! grep -q "worker_rlimit_nofile" /etc/nginx/nginx.conf; then
        sed -i '/^worker_processes/a worker_rlimit_nofile 65536;' /etc/nginx/nginx.conf
    fi
    sed -i 's/^\(\s*\)worker_connections 768;/\1worker_connections 4096;/' /etc/nginx/nginx.conf
fi

# Reload systemd (picks up nginx.service.d drop-in)
systemctl daemon-reload

# Test nginx config and restart to apply new limits + micropanel.conf
if command -v nginx >/dev/null 2>&1; then
    if nginx -t >/dev/null 2>&1; then
        systemctl restart nginx >/dev/null 2>&1 || true
    else
        echo "WARNING: nginx -t failed; skipping nginx restart. Run 'nginx -t' to debug."
    fi
fi

echo ""
echo "MicroPanel installed successfully!"
echo ""
echo "Next steps:"
echo "  1. Edit config: nano /etc/micropanel/config.yaml"
echo "     - Set panel_domain to your panel's domain (e.g., panel.example.com)"
echo ""
echo "  2. Create admin user:"
echo "     micropanel user create -e admin@example.com -p yourpassword -r admin"
echo ""
echo "  3. Setup nginx and SSL: /usr/share/micropanel/scripts/setup-panel-nginx.sh"
echo "     (Use --no-ssl flag to skip SSL)"
echo ""
echo "  4. Start service: systemctl enable --now micropanel"
echo ""
echo "NOTE: Service will not start until config and admin user are set up!"
echo ""
