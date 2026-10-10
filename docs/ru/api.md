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

Все маршруты требуют токена с `user_id`. Токен видит только сайты своего
пользователя; токен администратора — все сайты. Обращение к чужому сайту —
`403 Forbidden`.

### Объект сайта

Маршруты, возвращающие сайт, отвечают одним и тем же объектом:

```json
{
  "id": 1,
  "name": "example.com",
  "is_enabled": true,
  "ssl_enabled": true,
  "fix_mime_types": false,
  "ssl_status": "issued",
  "ssl_expires_at": "2026-12-31T10:00:00Z"
}
```

| Поле | Описание |
|------|----------|
| `ssl_enabled` | у сайта есть сертификат, и nginx отдаёт его по HTTPS |
| `ssl_status` | состояние последнего выпуска сертификата: `none` (не выпускался), `issuing` (идёт), `issued` (выпущен), `failed` (не удался) |
| `ssl_error` | почему последний выпуск не удался (только при `failed`; хвост вывода certbot, до 1000 символов) |
| `ssl_expires_at` | срок действия сертификата (RFC 3339), если он есть |

Выпуск, прерванный перезапуском панели, после старта становится `failed`
с причиной `certificate issuance interrupted by a panel restart`.

### Создание сайта

```
POST /api/v1/sites
```

**Тело запроса:**
```json
{
  "name": "example.com",
  "ssl": false,
  "fix_mime_types": false
}
```

| Параметр | Тип | Обязательный | По умолчанию | Описание |
|----------|-----|--------------|--------------|----------|
| name | string | да | - | Доменное имя сайта |
| ssl | bool | нет | false | Выпустить сертификат сразу после создания (синхронно); обычно сертификат заказывают отдельно через `POST /api/v1/sites/:id/ssl` |
| fix_mime_types | bool | нет | false | Исправлять MIME-типы файлов с закодированной строкой запроса в имени |

Сайт сразу становится живым: nginx отдаёт заглушку, пока не загружен архив.

**Ответ:** `201 Created` с объектом сайта. Если сайт с таким именем уже
существует, возвращается он же с `200 OK` — создание идемпотентно по имени.

> **Важно:** при совпадении имени возвращается существующий сайт, даже если
> он принадлежит другому пользователю. Проверяйте доступ запросом
> `GET /api/v1/sites/:id`: чужой сайт ответит `403`.

**Ошибки:**
- `400 Bad Request` — name не указан
- `401 Unauthorized` — неверный токен
- `403 Forbidden` — у токена нет `user_id`
- `500 Internal Server Error` — сайт не создан или конфигурация nginx не применилась

### Список сайтов

```
GET /api/v1/sites
```

**Ответ (200 OK):** массив объектов сайта.

### Информация о сайте

```
GET /api/v1/sites/:id
```

**Ответ (200 OK):** объект сайта.

**Ошибки:**
- `400 Bad Request` — неверный ID
- `403 Forbidden` — чужой сайт
- `404 Not Found` — сайт не найден

### Удаление сайта

```
DELETE /api/v1/sites/:id
```

Удаляет сертификат (если есть), конфигурацию nginx и сайт.

**Ответ (200 OK):**
```json
{
  "message": "site deleted"
}
```

**Ошибки:**
- `400 Bad Request` — неверный ID
- `403 Forbidden` — чужой сайт
- `404 Not Found` — сайт не найден

### Деплой архива

```
POST /api/v1/sites/:id/deploy
```

**Content-Type:** `multipart/form-data`

**Параметры:**
- `file` — архив (`.zip`, `.tar.gz` или `.tgz`). Если у всех файлов архива
  общий корневой каталог, он срезается.

**Заголовки:**
- `Idempotency-Key` (необязательный) — ключ безопасного повтора: 1–255
  видимых символов ASCII, уникален в пределах сайта.

Архив распаковывается рядом с текущей версией и подменяет её двумя
переименованиями; прежняя версия остаётся в `public_prev`. Деплои одного
сайта выполняются по очереди.

**Ответ (200 OK):**
```json
{
  "deploy_id": 1,
  "status": "success"
}
```

**Повтор с `Idempotency-Key`:**

| Деплой с этим ключом | Ответ |
|----------------------|-------|
| нет | обычный деплой |
| успешный | `200 OK` с прежним `deploy_id` и `"replayed": true`; архив не распаковывается, `public_prev` не меняется |
| ещё идёт | `409 Conflict` с `deploy_id` идущего деплоя |
| неудачный | новая попытка в той же записи (тот же `deploy_id`) |

Без заголовка каждый запрос — новый деплой. Деплой, прерванный перезапуском
панели, после старта становится `failed`, и его ключ можно повторить.

**Ошибки:**
- `400 Bad Request` — файл не указан, неверный `Idempotency-Key`, неверный формат архива, слишком много файлов, выход за пределы каталога, символические ссылки
- `403 Forbidden` — чужой сайт
- `404 Not Found` — сайт не найден
- `409 Conflict` — деплой с этим `Idempotency-Key` ещё идёт
- `413 Request Entity Too Large` — архив больше 100 МБ

Пределы архива зашиты в код: архив до 100 МБ, файл до 10 МБ, до 10 000
файлов, путь до 500 символов, путь с `..` отклоняется.

### Список деплоев

```
GET /api/v1/sites/:id/deploys
```

**Параметры запроса:**

| Параметр | По умолчанию | Описание |
|----------|--------------|----------|
| limit | 20 | сколько последних деплоев вернуть, не больше 100 |
| idempotency_key | - | вернуть только деплой с этим ключом (пустой массив, если его нет) |

**Ответ (200 OK):** деплои от новых к старым.
```json
[
  {
    "id": 2,
    "status": "failed",
    "idempotency_key": "publication-7",
    "error": "extract archive: ...",
    "created_at": "2026-10-10T12:00:00Z"
  }
]
```

`status` — `pending`, `success` или `failed`.

**Ошибки:**
- `400 Bad Request` — неверный ID или limit
- `403 Forbidden` — чужой сайт
- `404 Not Found` — сайт не найден

### Выпуск сертификата

```
POST /api/v1/sites/:id/ssl
```

Сертификат Let's Encrypt через certbot (`--webroot`). Имена сайта должны уже
указывать на сервер панели.

**Тело запроса (необязательное):**
```json
{
  "mode": "all",
  "async": true
}
```

| Параметр | По умолчанию | Описание |
|----------|--------------|----------|
| mode | `all` | `all` — имя сайта, `www.` (если включён) и алиасы; `primary` — имя сайта и `www.`; `aliases` — только алиасы; `none` — ничего не выпускать |
| async | false | выпускать в фоне |

**Синхронно** (`async` не задан): ответ после окончания certbot (до 5
минут) — `200 OK` с объектом сайта или `500` при неудаче.

**Асинхронно** (`"async": true`): сразу `202 Accepted` с объектом сайта в
состоянии `ssl_status: "issuing"`; итог читается из `GET /api/v1/sites/:id`
(`issued` или `failed` с `ssl_error`). Повторный запрос, пока выпуск идёт, —
тоже `202`, второй certbot не запускается. Выпуски всех сайтов панели идут
по одному.

**Ошибки:**
- `400 Bad Request` — неверный ID, неверный mode, нет имён для выпуска
- `403 Forbidden` — чужой сайт
- `404 Not Found` — сайт не найден
- `500 Internal Server Error` — синхронный выпуск не удался

### Алиасы сайта

Дополнительные имена, которые обслуживает тот же сайт.

```
GET /api/v1/sites/:id/domains
POST /api/v1/sites/:id/domains
DELETE /api/v1/sites/:id/domains/:domainId
```

`POST` принимает `{"hostname": "www.example.org"}` и отвечает `201 Created`:

```json
{
  "id": 3,
  "site_id": 1,
  "hostname": "www.example.org"
}
```

Повторное добавление того же имени к тому же сайту — `200 OK` с
существующим алиасом; имя, занятое другим сайтом, — `409 Conflict`. Имя
сайта и его `www.` алиасом не добавляются (`400`). `GET` возвращает массив
алиасов, `DELETE` — `{"message": "domain alias deleted"}`. Сертификат на
алиасы — `POST /api/v1/sites/:id/ssl` с `mode` `all` или `aliases`.

## Примеры использования

### cURL

```bash
# Создать сайт (без сертификата — по умолчанию)
curl -X POST http://localhost:8080/api/v1/sites \
  -H "Authorization: Bearer your-secret-token" \
  -H "Content-Type: application/json" \
  -d '{"name": "example.com"}'

# Заказать сертификат в фоне и узнать итог
curl -X POST http://localhost:8080/api/v1/sites/1/ssl \
  -H "Authorization: Bearer your-secret-token" \
  -H "Content-Type: application/json" \
  -d '{"mode": "primary", "async": true}'
curl http://localhost:8080/api/v1/sites/1 \
  -H "Authorization: Bearer your-secret-token"

# Список сайтов
curl http://localhost:8080/api/v1/sites \
  -H "Authorization: Bearer your-secret-token"

# Информация о сайте
curl http://localhost:8080/api/v1/sites/1 \
  -H "Authorization: Bearer your-secret-token"

# Загрузить архив (повтор с тем же ключом не распакует его заново)
curl -X POST http://localhost:8080/api/v1/sites/1/deploy \
  -H "Authorization: Bearer your-secret-token" \
  -H "Idempotency-Key: release-42" \
  -F "file=@site.zip"

# Последние деплои
curl "http://localhost:8080/api/v1/sites/1/deploys?limit=5" \
  -H "Authorization: Bearer your-secret-token"

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
| 202 | Принято, выполняется в фоне |
| 400 | Неверный запрос |
| 401 | Не авторизован |
| 403 | Доступ запрещен (IP не в whitelist, чужой сайт, токен без user_id) |
| 404 | Ресурс не найден |
| 409 | Конфликт (алиас занят другим сайтом, деплой с этим ключом ещё идёт) |
| 413 | Слишком большой запрос |
| 429 | Слишком много запросов |
| 500 | Внутренняя ошибка сервера |
