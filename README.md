# URL Shortener

Сервис сокращения URL с аналитикой переходов, реализованный на Go с использованием Clean Architecture.

## Возможности

- Создание коротких ссылок (POST /shorten)
- Редирект по коротким ссылкам (GET /s/{short_url})
- Аналитика переходов (GET /analytics/{short_url})
- Кастомные алиасы для ссылок
- Кэширование через Redis (опционально)
- Простой веб-интерфейс для тестирования

## Архитектура

Проект следует принципам Clean Architecture и Standard Go Project Layout:

```
├── cmd/server/          # Точка входа приложения
├── internal/
│   ├── domain/         # Доменный слой (entities, repositories interfaces, services)
│   ├── application/    # Слой приложения (use cases)
│   └── infrastructure/ # Слой инфраструктуры (БД, HTTP, кэш)
└── web/                # Веб-интерфейс
```

## Требования

- Go 1.21+
- PostgreSQL 15+
- Redis 7+ (опционально, для кэширования)

## Установка и запуск

### 1. Клонирование и установка зависимостей

```bash
go mod download
```

### 2. Настройка переменных окружения

Скопируйте пример файла конфигурации и настройте под свои нужды:

```bash
cp .env.example .env
```

Отредактируйте `.env` файл с вашими настройками:

```env
PORT=8080
BASE_URL=http://localhost:8080
DATABASE_DSN=postgres://user:password@localhost/shortener?sslmode=disable
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
ENABLE_REDIS=false
REDIS_TTL=30m
```

**Приоритет конфигурации:**
1. Флаги командной строки (высший приоритет)
2. Переменные окружения (из `.env` файла или системные)
3. Значения по умолчанию

### 3. Запуск PostgreSQL и Redis через Docker Compose

```bash
docker-compose up -d
```

Docker Compose автоматически использует переменные окружения из `.env` файла (если они заданы) или значения по умолчанию.

### 4. Запуск сервера

**Простой запуск (использует .env файл):**
```bash
go run cmd/server/main.go
```

**Или через Makefile:**
```bash
make run          # Запуск без Redis
make run-redis    # Запуск с Redis
```

**С переопределением через флаги:**
```bash
go run cmd/server/main.go \
  -port=8080 \
  -db="postgres://user:password@localhost/shortener?sslmode=disable" \
  -base-url="http://localhost:8080" \
  -enable-redis
```

**Или через переменные окружения:**
```bash
export PORT=8080
export DATABASE_DSN="postgres://user:password@localhost/shortener?sslmode=disable"
export ENABLE_REDIS=true
go run cmd/server/main.go
```

### 5. Открыть веб-интерфейс

Откройте браузер и перейдите по адресу: http://localhost:8080

## API Эндпоинты

### POST /shorten

Создание новой короткой ссылки.

**Запрос:**
```json
{
  "original_url": "https://example.com/very/long/url",
  "custom_alias": "my-link"  // опционально
}
```

**Успешный ответ (201 Created):**
```json
{
  "short_url": "http://localhost:8080/s/abc123",
  "original_url": "https://example.com/very/long/url"
}
```

**Ошибки:**
- `400 Bad Request` - неверный формат запроса или URL
- `409 Conflict` - кастомный алиас уже существует
- `500 Internal Server Error` - внутренняя ошибка сервера

**Валидация:**
- `original_url` обязателен
- URL должен иметь схему `http://` или `https://`
- URL должен содержать валидный хост
- Максимальная длина URL: 2048 символов

### GET /s/{short_url}

Редирект на оригинальный URL. Автоматически регистрирует переход.

**Ошибки:**
- `400 Bad Request` - неверный формат короткого URL
- `404 Not Found` - ссылка не найдена
- `500 Internal Server Error` - внутренняя ошибка сервера

### GET /analytics/{short_url}

Получение аналитики по короткой ссылке.

**Ошибки:**
- `400 Bad Request` - неверный формат короткого URL
- `404 Not Found` - ссылка не найдена
- `500 Internal Server Error` - внутренняя ошибка сервера

**Ответ:**
```json
{
  "link_id": 1,
  "short_url": "abc123",
  "total_clicks": 42,
  "by_day": {
    "2024-01-15": 10,
    "2024-01-16": 32
  },
  "by_month": {
    "2024-01": 42
  },
  "by_user_agent": {
    "Mozilla/5.0...": 30,
    "curl/7.68.0": 12
  },
  "recent_clicks": [
    {
      "id": 1,
      "link_id": 1,
      "user_agent": "Mozilla/5.0...",
      "ip_address": "127.0.0.1",
      "clicked_at": "2024-01-16T10:30:00Z"
    }
  ]
}
```

## Формат ответов об ошибках

Все ошибки возвращаются в структурированном формате:

```json
{
  "error": "Описание ошибки",
  "code": "код_ошибки",
  "message": "Описание ошибки"
}
```

**Коды ошибок:**
- `invalid_request_body` - неверный формат тела запроса
- `url_required` - не указан original_url
- `invalid_url` - неверный формат URL
- `alias_exists` - кастомный алиас уже существует
- `link_not_found` - ссылка не найдена
- `invalid_short_url` - неверный формат короткого URL
- `method_not_allowed` - неверный HTTP метод
- `internal_error` - внутренняя ошибка сервера

## Примеры использования

### Создание ссылки через curl

```bash
curl -X POST http://localhost:8080/shorten \
  -H "Content-Type: application/json" \
  -d '{"original_url": "https://example.com"}'
```

### Создание ссылки с кастомным алиасом

```bash
curl -X POST http://localhost:8080/shorten \
  -H "Content-Type: application/json" \
  -d '{"original_url": "https://example.com", "custom_alias": "my-link"}'
```

### Получение аналитики

```bash
curl http://localhost:8080/analytics/abc123
```

### Пример ответа об ошибке

```bash
curl -X POST http://localhost:8080/shorten \
  -H "Content-Type: application/json" \
  -d '{"original_url": "invalid-url"}'
```

**Ответ (400 Bad Request):**
```json
{
  "error": "Invalid URL format",
  "code": "invalid_url",
  "message": "Invalid URL format"
}
```

## Технологии

- **Go** - основной язык программирования
- **PostgreSQL** - база данных
- **Redis** - кэширование (опционально)
- **Clean Architecture** - архитектурный подход
- **SOLID принципы** - проектирование кода

## Особенности реализации

- Типизированные ошибки для надежной обработки
- Структурированное логирование
- Валидация входных данных (URL, размер запроса)
- Централизованная обработка ошибок
- Защита от переполнения (ограничение размера тела запроса)
- Graceful shutdown для корректного завершения работы

## Структура проекта

- `internal/domain/` - доменные сущности и интерфейсы
- `internal/application/` - бизнес-логика (use cases)
- `internal/infrastructure/` - реализация инфраструктуры (БД, HTTP, кэш)
- `cmd/server/` - точка входа приложения
- `web/` - веб-интерфейс

## Лицензия

MIT

