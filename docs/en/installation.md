# MicroPanel Installation

## Requirements

- Ubuntu 20.04+ / Debian 11+ / CentOS 8+
- Nginx
- Root access

## Package Installation (Recommended)

### Debian/Ubuntu (APT)

```bash
# Add repository
echo "deb [trusted=yes] https://foxzi.github.io/micropanel/apt stable main" | sudo tee /etc/apt/sources.list.d/micropanel.list

# Install package
sudo apt update
sudo apt install micropanel
```

### CentOS/RHEL (RPM)

```bash
# Download and install package
sudo dnf install https://foxzi.github.io/micropanel/rpm/micropanel-1.2.0-1.x86_64.rpm
```

After installation, complete the setup (see below).

## Post-Installation Setup

```bash
# 1. Set panel domain
sudo nano /etc/micropanel/config.yaml
# Set panel_domain: panel.example.com

# 2. Create admin user
sudo micropanel user create -e admin@example.com -p yourpassword -r admin

# 3. Setup nginx and SSL
sudo /usr/share/micropanel/scripts/setup-panel-nginx.sh
# Or without SSL: sudo /usr/share/micropanel/scripts/setup-panel-nginx.sh --no-ssl

# 4. Start service
sudo systemctl enable --now micropanel
```

**Important:** Service will not start until steps 1 and 2 are completed!

## Configuration

Configuration file: `/etc/micropanel/config.yaml`

```yaml
app:
  env: production
  host: 127.0.0.1
  port: 8080
  secret: auto-generated-on-install
  panel_domain: panel.example.com

database:
  path: /var/lib/micropanel/micropanel.db

sites:
  path: /var/www/panel/sites

nginx:
  config_path: /etc/nginx/sites-enabled
  reload_cmd: sudo nginx -s reload

ssl:
  email: admin@example.com
  staging: false

api:
  enabled: false
  tokens: []

security:
  panel_allowed_ips: []
  api_allowed_ips: []
```

## Installation Paths

| Path | Description |
|------|-------------|
| `/usr/bin/micropanel` | Binary |
| `/etc/micropanel/config.yaml` | Configuration |
| `/var/lib/micropanel/` | Database |
| `/var/www/panel/sites/` | Site files |
| `/usr/share/micropanel/` | Migrations, scripts, static |

## Docker (for development)

```bash
git clone https://github.com/foxzi/micropanel.git
cd micropanel
docker compose up -d
```

Panel available at http://localhost:8081

## Service Management

```bash
# Status
sudo systemctl status micropanel

# Restart
sudo systemctl restart micropanel

# Logs
sudo journalctl -u micropanel -f
```
