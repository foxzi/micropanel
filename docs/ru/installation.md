# Установка MicroPanel

## Требования

- Ubuntu 20.04+ / Debian 11+ / CentOS 8+
- Nginx
- Root доступ

## Установка из пакета (рекомендуется)

### Debian/Ubuntu (APT)

```bash
# Добавьте репозиторий
echo "deb [trusted=yes] https://foxzi.github.io/micropanel/apt stable main" | sudo tee /etc/apt/sources.list.d/micropanel.list

# Установите пакет
sudo apt update
sudo apt install micropanel

# Запустите сервис
sudo systemctl enable --now micropanel
```

### CentOS/RHEL (RPM)

```bash
# Скачайте и установите пакет
sudo dnf install https://foxzi.github.io/micropanel/rpm/micropanel-1.2.0-1.x86_64.rpm

# Запустите сервис
sudo systemctl enable --now micropanel
```

## Настройка после установки

```bash
# 1. Укажите домен панели
sudo nano /etc/micropanel/config.yaml
# Установите panel_domain: panel.example.com

# 2. Настройте nginx
sudo /usr/share/micropanel/scripts/setup-panel-nginx.sh

# 3. Запустите сервис
sudo systemctl enable --now micropanel

# 4. (Опционально) Включите SSL
sudo certbot --nginx -d panel.example.com
```

## Учетные данные по умолчанию

- **Email:** admin@localhost
- **Пароль:** admin

**Важно:** Измените пароль после первого входа!

## Конфигурация

Конфигурационный файл: `/etc/micropanel/config.yaml`

```yaml
app:
  env: production
  port: 8080
  secret: auto-generated-on-install

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

## Пути установки

| Путь | Описание |
|------|----------|
| `/usr/bin/micropanel` | Бинарный файл |
| `/etc/micropanel/config.yaml` | Конфигурация |
| `/var/lib/micropanel/` | База данных |
| `/var/www/panel/sites/` | Файлы сайтов |
| `/usr/share/micropanel/` | Миграции, скрипты, статика |

## Docker (для разработки)

```bash
git clone https://github.com/foxzi/micropanel.git
cd micropanel
docker compose up -d
```

Панель доступна на http://localhost:8081

## Управление сервисом

```bash
# Статус
sudo systemctl status micropanel

# Перезапуск
sudo systemctl restart micropanel

# Логи
sudo journalctl -u micropanel -f
```
