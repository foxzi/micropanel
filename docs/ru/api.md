# REST API

MicroPanel предоставляет REST API для автоматизации управления сайтами и деплоя.

## Включение API

Добавьте в `config.yaml`:

```yaml
api:
  enabled: true
  tokens: []  # Токены можно создавать через веб-панель
```

Или через переменную окружения:

```bash
API_ENABLED=true
```

## Управление токенами

### Через веб-панель (рекомендуется)

Все пользователи могут создавать и управлять своими API-токенами через веб-панель:
1. Войдите в панель
2. Перейдите в раздел "API Tokens" в навигации
3. Нажмите "Create Token" и введите название
4. Скопируйте токен (он показывается только один раз!)

Токены привязаны к пользователю и дают доступ только к его сайтам.

### Через конфиг (для обратной совместимости)

Можно также добавить токены в `config.yaml`:

```yaml
api:
  enabled: true
  tokens:
    - name: "deploy-bot"
      token: "your-secret-token-here"
      user_id: 1  # ID пользователя-владельца
    - name: "ci-cd"
      token: "another-secret-token"
      user_id: 2
```

> **Важно:** Поле `user_id` обязательно. Токен без `user_id` будет отклонен.

## Аутентификация

API использует Bearer-токены. Добавляйте заголовок `Authorization` к каждому запросу:

```
Authorization: Bearer your-secret-token-here
```

## Ограничение доступа по IP

Можно ограничить доступ к API по IP-адресам. Поддерживается CIDR-нотация:

```yaml
security:
  # Whitelist для веб-панели (пустой список = доступ всем)
  panel_allowed_ips:
    - "192.168.1.0/24"
    - "10.0.0.1"

  # Whitelist для API (пустой список = доступ всем)
  api_allowed_ips:
    - "192.168.1.100"
    - "10.0.0.0/8"
```

## Endpoints

### Создание сайта

```
POST /api/v1/sites
```

**Тело запроса:**
```json
{
  "name": "example.com",
  "ssl": true
}
```

**Параметры:**
| Параметр | Тип | Обязательный | По умолчанию | Описание |
|----------|-----|--------------|--------------|----------|
| name | string | да | - | Доменное имя сайта |
| ssl | bool | нет | true | Автоматически выпустить SSL-сертификат |

**Ответ (201 Created):**
```json
{
  "id": 1,
  "name": "example.com",
  "is_enabled": true,
  "ssl_enabled": false
}
```

> **Примечание:** `ssl_enabled` в ответе показывает текущий статус. Сертификат выпускается асинхронно, поэтому сразу после создания будет `false`. Статус обновится после успешного выпуска сертификата.

**Ошибки:**
- `400 Bad Request` - name не указан
- `401 Unauthorized` - неверный токен
- `409 Conflict` - сайт с таким именем уже существует

### Список сайтов

```
GET /api/v1/sites
```

**Ответ (200 OK):**
```json
[
  {
    "id": 1,
    "name": "example.com",
    "is_enabled": true,
    "ssl_enabled": true
  },
  {
    "id": 2,
    "name": "test.com",
    "is_enabled": true,
    "ssl_enabled": false
  }
]
```

### Информация о сайте

```
GET /api/v1/sites/:id
```

**Ответ (200 OK):**
```json
{
  "id": 1,
  "name": "example.com",
  "is_enabled": true,
  "ssl_enabled": true
}
```

**Ошибки:**
- `400 Bad Request` - неверный ID
- `404 Not Found` - сайт не найден

### Удаление сайта

```
DELETE /api/v1/sites/:id
```

**Ответ (200 OK):**
```json
{
  "message": "site deleted"
}
```

**Ошибки:**
- `400 Bad Request` - неверный ID
- `404 Not Found` - сайт не найден

### Деплой архива

```
POST /api/v1/sites/:id/deploy
```

**Content-Type:** `multipart/form-data`

**Параметры:**
- `file` - архив (ZIP или TAR.GZ)

**Ответ (200 OK):**
```json
{
  "deploy_id": 1,
  "status": "success"
}
```

**Ошибки:**
- `400 Bad Request` - файл не указан или неверный формат
- `404 Not Found` - сайт не найден
- `413 Request Entity Too Large` - архив слишком большой (макс. 100MB)

## Примеры использования

### cURL

```bash
# Создать сайт с SSL (по умолчанию)
curl -X POST http://localhost:8080/api/v1/sites \
  -H "Authorization: Bearer your-secret-token" \
  -H "Content-Type: application/json" \
  -d '{"name": "example.com"}'

# Создать сайт без SSL
curl -X POST http://localhost:8080/api/v1/sites \
  -H "Authorization: Bearer your-secret-token" \
  -H "Content-Type: application/json" \
  -d '{"name": "example.com", "ssl": false}'

# Список сайтов
curl http://localhost:8080/api/v1/sites \
  -H "Authorization: Bearer your-secret-token"

# Информация о сайте
curl http://localhost:8080/api/v1/sites/1 \
  -H "Authorization: Bearer your-secret-token"

# Загрузить архив
curl -X POST http://localhost:8080/api/v1/sites/1/deploy \
  -H "Authorization: Bearer your-secret-token" \
  -F "file=@site.zip"

# Удалить сайт
curl -X DELETE http://localhost:8080/api/v1/sites/1 \
  -H "Authorization: Bearer your-secret-token"
```

### GitHub Actions

```yaml
name: Deploy

on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Create archive
        run: zip -r site.zip . -x ".git/*"

      - name: Deploy to MicroPanel
        run: |
          curl -X POST ${{ secrets.MICROPANEL_URL }}/api/v1/sites/${{ secrets.SITE_ID }}/deploy \
            -H "Authorization: Bearer ${{ secrets.MICROPANEL_TOKEN }}" \
            -F "file=@site.zip"
```

### Python

```python
import requests

API_URL = "http://localhost:8080/api/v1"
TOKEN = "your-secret-token"

headers = {"Authorization": f"Bearer {TOKEN}"}

# Создать сайт
response = requests.post(
    f"{API_URL}/sites",
    headers=headers,
    json={"name": "example.com"}
)
site = response.json()
print(f"Created site: {site['id']}")

# Деплой
with open("site.zip", "rb") as f:
    response = requests.post(
        f"{API_URL}/sites/{site['id']}/deploy",
        headers=headers,
        files={"file": f}
    )
print(response.json())
```

## Rate Limiting

API ограничен 100 запросами в минуту на IP-адрес.

## Коды ответов

| Код | Описание |
|-----|----------|
| 200 | Успешный запрос |
| 201 | Ресурс создан |
| 400 | Неверный запрос |
| 401 | Не авторизован |
| 403 | Доступ запрещен (IP не в whitelist) |
| 404 | Ресурс не найден |
| 409 | Конфликт (ресурс уже существует) |
| 413 | Слишком большой запрос |
| 429 | Слишком много запросов |
| 500 | Внутренняя ошибка сервера |
