# realtime-chat-app

<img src="https://static.wikia.nocookie.net/logopedia/images/9/94/VK_Education.svg/revision/latest?cb=20230429123445" width="150" alt="VK Education">

Мессенджер с чатами 1-на-1 и групповыми, доставкой сообщений в реальном времени через WebSocket, историей с пагинацией и поиском по тексту.

Бэкенд на Go, фронтенд на React.

## Структура репозитория

- `backend/` - Go API (REST + WebSocket)
- `frontend/` - React + TypeScript SPA
- `docker-compose.yml` - Postgres для локальной разработки

## Стек

### Backend

- Go 1.26
- PostgreSQL, драйвер `jackc/pgx/v5`
- JWT (`golang-jwt/jwt/v5`: access + refresh токены) и `bcrypt` (`golang.org/x/crypto`) для хэширования паролей
- WebSocket (`gorilla/websocket`)
- `net/http` (стандартный роутер stdlib, без фреймворка)
- `google/uuid` для идентификаторов сущностей

Слойная архитектура: `transport (http/websocket) -> service -> repository -> database`.

- `internal/models` - доменные сущности
- `internal/repository` - доступ к Postgres
- `internal/service` - бизнес-логика
- `internal/transport/http` - REST API
- `internal/transport/websocket` - WebSocket
- `internal/middleware` - JWT middleware
- `internal/security` - bcrypt, JWT, refresh-токены
- `internal/database` - подключение к Postgres и SQL-миграции
- `internal/app` - сборка приложения 
- `cmd/api` - точка входа

### Frontend

- React 18 + TypeScript
- Vite (сборка и dev-сервер с прокси `/api` -> `http://localhost:8080`)
- React Router (`react-router-dom`) для роутинга по экранам
- Обычный `fetch` через самописный API-клиент (`src/api.ts`), без внешних HTTP/state-библиотек
- Нативный `WebSocket` для подписки на сообщения чата в реальном времени

## Запуск проекта

Требования: Go 1.26+, Node.js 18+, Docker (для Postgres).

### 1. База данных

```
docker compose up -d
```

Поднимет Postgres на `localhost:5433` (см. `docker-compose.yml`), БД `realtime_chat`.

Накатить миграции из `backend/internal/database/migrations` вручную через `psql` (по порядку файлов `001_...` -> `006_...`, затем `indexes.sql`), например:

```
for f in backend/internal/database/migrations/*.sql; do
  psql "postgresql://postgres:postgres@localhost:5433/realtime_chat?sslmode=disable" -f "$f"
done
```

### 2. Backend

Создать `backend/.env` (или экспортировать переменные окружения):

```
DB_ADDRESS=postgresql://postgres:postgres@localhost:5433/realtime_chat?sslmode=disable
JWT_SECRET=<любая секретная строка>
```

Дополнительные переменные (не обязательны, есть значения по умолчанию):

- `SERVER_PORT` - порт HTTP-сервера (по умолчанию `8080`)
- `TOKEN_TTL` - время жизни access-токена (по умолчанию `24h`)
- `REFRESH_TOKEN_TTL` - время жизни refresh-токена (по умолчанию `168h`)

Запуск:

```
cd backend
go run ./cmd/api
```

Проверка, что сервер поднялся: `GET http://localhost:8080/healthz` - бэкенд отдаст "ok".

### 3. Frontend

```
cd frontend
npm install
npm run dev
```

Откроется на `http://localhost:5173`, запросы к `/api/*` проксируются на бэкенд на `8080`.

### Тесты

```
cd backend
go test ./...
```

## API

Базовый префикс: `/api/v1`. Все ручки, кроме `/auth/*`, требуют заголовок `Authorization: Bearer <access_token>`.

**Auth:** `POST /auth/register`, `POST /auth/login`, `POST /auth/refresh`, `POST /auth/logout`.

**Чаты:** `GET /chats`, `POST /chats`, `DELETE /chats/{chat_id}`, `GET /chats/{chat_id}/members`, `POST /chats/{chat_id}/members`, `DELETE /chats/{chat_id}/members/{user_id}` 

**Сообщения:** `GET /chats/{chat_id}/messages?limit=&offset=` (история, по умолчанию `limit=50`, максимум `100`), `POST /chats/{chat_id}/messages` (отправка через REST, без WS), `GET /chats/{chat_id}/search?q=` (поиск по тексту, `ILIKE`).

### WebSocket

`GET /api/v1/chats/{chat_id}/ws?token=<access_token>&after_seq=<seq>`

Токен передаётся query-параметром (браузерный `WebSocket` API не позволяет ставить заголовки). Перед апгрейдом сервер проверяет токен и членство пользователя в чате. Параметр `after_seq` опционален - при реконнекте сервер реплеит пропущенные сообщения (`seq > after_seq`), после чего клиент переходит в live-режим.

Входящий фрейм от клиента:

```json
{ "body": "текст сообщения", "client_msg_id": "уникальный ID от клиента" }
```

Исходящие фреймы различаются полем `type`: `"message"` (новое сообщение, рассылается всем в чате), `"ack"` (подтверждение отправителю, что сообщение принято и сохранено), `"error"`.

Надёжность доставки:

- **Идемпотентность** - уникальный индекс `(chat_id, client_msg_id)` в БД: повторная отправка того же `client_msg_id` не создаёт дубликат, сервер просто вернёт `ack` на уже существующее сообщение.
- **Ack + retry** - сервер шлёт `ack` на каждое сообщение; если клиент не дождался `ack` за таймаут, он повторяет отправку с тем же `client_msg_id`.
- **Порядок** - у каждого сообщения есть монотонно возрастающий `seq` (`BIGSERIAL`), не завязанный на `created_at`.
