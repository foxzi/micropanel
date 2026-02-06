# Установка MicroPanel

## Требования

- Docker и Docker Compose
- Git

## Быстрый старт

```bash
# Клонируйте репозиторий
git clone https://github.com/yourname/micropanel.git
cd micropanel

# Запустите с Docker Compose
make dev

# Откройте в браузере
open http://localhost:8081
```

## Учетные данные по умолчанию

- **Email:** admin@localhost
- **Пароль:** admin

**Важно:** Измените пароль после первого входа!

## Конфигурация

### Переменные окружения

| Переменная | По умолчанию | Описание |
|------------|--------------|----------|
| `APP_ENV` | `development` | Окружение (development/production) |
| `APP_PORT` | `8080` | Порт приложения |
| `APP_SECRET` | - | Секретный ключ сессий |
| `DB_PATH` | `./data/micropanel.db` | Путь к базе данных SQLite |
| `SITES_PATH` | `/var/www/panel/sites` | Корневая директория сайтов |

### Файл конфигурации

Скопируйте `config.yaml.example` в `config.yaml` и настройте:

```yaml
app:
  env: production
  port: 8080
  secret: ваш-секретный-ключ

database:
  path: /app/data/micropanel.db

sites:
  path: /var/www/panel/sites

nginx:
  config_path: /etc/nginx/sites-enabled
  reload_cmd: nginx -s reload
```

## Структура директорий

```
micropanel/
├── cmd/micropanel/     # Точка входа
├── internal/           # Внутренние пакеты
│   ├── config/         # Конфигурация
│   ├── database/       # База данных
│   ├── handlers/       # HTTP обработчики
│   ├── middleware/     # Middleware
│   ├── models/         # Модели данных
│   ├── repository/     # Репозитории
│   ├── services/       # Бизнес-логика
│   └── templates/      # Templ шаблоны
├── migrations/         # SQL миграции
├── web/static/         # Статические файлы
├── docker/             # Docker файлы
└── docs/               # Документация
```
