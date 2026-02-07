#!/bin/bash
set -e

# Set ownership
chown -R micropanel:micropanel /var/lib/micropanel
chown -R micropanel:micropanel /var/www/panel/sites
chown root:micropanel /etc/micropanel/config.yaml
chmod 640 /etc/micropanel/config.yaml

# Create sudoers file for micropanel user
cat > /etc/sudoers.d/micropanel <<EOF
# Allow micropanel to run certbot and nginx without password
micropanel ALL=(ALL) NOPASSWD: /usr/bin/certbot
micropanel ALL=(ALL) NOPASSWD: /usr/sbin/nginx
EOF
chmod 440 /etc/sudoers.d/micropanel

# Generate random secret if default is present
if grep -q "change-me-min-32-chars" /etc/micropanel/config.yaml 2>/dev/null; then
    SECRET=$(openssl rand -hex 32)
    sed -i "s/change-me-min-32-chars-random-string/$SECRET/" /etc/micropanel/config.yaml
fi

# Reload systemd
systemctl daemon-reload

echo ""
echo "MicroPanel installed successfully!"
echo ""
echo "Next steps:"
echo "  1. Edit config: nano /etc/micropanel/config.yaml"
echo "  2. Start service: systemctl enable --now micropanel"
echo "  3. Access panel at http://localhost:8080"
echo "  4. Default login: admin@localhost / admin"
echo ""
