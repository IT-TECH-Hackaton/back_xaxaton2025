# Техническое задание: Проект MyAfisha

## 1. Общее описание

**MyAfisha** — веб-приложение «Электронная афиша» для управления мероприятиями, регистрации участников, поиска компании и формирования микросообществ по интересам.

**Цель ТЗ:** дать разработчику полную инструкцию для воспроизведения проекта с нуля.

---

## 2. Архитектура

| Компонент | Технологии | Репозиторий |
|-----------|------------|-------------|
| Backend | Go 1.24, Gin, GORM, PostgreSQL 15 | `back_xaxaton2025` |
| Frontend | React 18, TypeScript, Vite, Tailwind, TanStack Query | `frontend-MyAfisha` |
| БД | PostgreSQL 15 | — |
| Оркестрация | Docker Compose | корневой `docker-compose.yml` |

---

## 3. Backend (back_xaxaton2025)

### 3.1 Структура проекта

```
back_xaxaton2025/
├── main.go                 # Точка входа, init admin, RunSeed
├── go.mod, go.sum
├── Dockerfile
├── .env                     # Конфигурация (не в git)
├── env.template
├── config/config.go
├── database/database.go
├── models/                  # User, Event, Category, Interest, Community, Matching, Review...
├── handlers/                # auth, event, user, admin, discover, community, matching...
├── middleware/              # auth, ratelimit
├── routes/routes.go
├── services/               # email, cron, seed
├── dto/
├── utils/
├── logger/
├── docs/                    # Swagger
├── scripts/                # seed_events, seed_users, create_admin
└── uploads/events/         # Изображения мероприятий (10 шт.)
```

### 3.2 Зависимости (go.mod)

- `github.com/gin-gonic/gin`
- `gorm.io/gorm`, `gorm.io/driver/postgres`
- `github.com/golang-jwt/jwt/v5`
- `github.com/google/uuid`
- `gopkg.in/gomail.v2`
- `github.com/robfig/cron/v3`
- `github.com/xuri/excelize/v2`
- `github.com/joho/godotenv`
- `go.uber.org/zap`
- `github.com/swaggo/gin-swagger`, `github.com/swaggo/files`

### 3.3 Переменные окружения (.env)

```
APP_PORT=8080
APP_ENV=development
DB_HOST=postgres
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=bekend
JWT_SECRET=<обязательно, минимум 32 символа>
JWT_EXPIRATION=24h
EMAIL_HOST=smtp.yandex.ru
EMAIL_PORT=465
EMAIL_USER=
EMAIL_PASSWORD=
EMAIL_FROM=
FRONTEND_URL=http://localhost:5173
CORS_ALLOW_ORIGINS=http://localhost:5173
YANDEX_CLIENT_ID=
YANDEX_CLIENT_SECRET=
YANDEX_REDIRECT_URI=http://localhost:8081/api/auth/yandex/callback
FAKE_YANDEX_AUTH=true
YANDEX_GEOCODER_API_KEY=
```

### 3.4 Основные сущности БД

- **users** — пользователи (роль, статус, email_verified, auth_provider)
- **events** — мероприятия (title, dates, imageURL, maxParticipants, status, organizer_id)
- **event_participants** — участники событий
- **event_reviews** — отзывы (rating, comment)
- **categories** — категории событий
- **interests** — интересы пользователей
- **user_interests** — связь пользователь–интерес (weight)
- **event_matchings** — «ищу компанию» на событии
- **match_requests** — запросы на знакомство (pending/accepted/rejected)
- **micro_communities**, **community_members** — микросообщества

### 3.5 API (основные группы)

| Группа | Маршруты |
|--------|----------|
| `/health` | GET — проверка работы |
| `/swagger/*` | Swagger UI |
| `/uploads/*` | Статика (аватарки, изображения событий) |
| `/api/auth/` | register, verify-email, login, logout, forgot/reset-password, yandex |
| `/api/upload/` | POST image |
| `/api/user/` | GET/PUT profile |
| `/api/events/` | CRUD, join, leave, export |
| `/api/events/hot` | GET — горящие события (72ч, 50%+ заполнены) |
| `/api/events/recommended` | GET — рекомендации по интересам (auth) |
| `/api/events/:id/reviews/` | CRUD отзывов |
| `/api/geocoder/` | geocode, reverse, map-link |
| `/api/interests/` | CRUD интересов |
| `/api/events/:id/matching/` | матчинг, запросы, accept/reject |
| `/api/communities/` | CRUD, join, leave |
| `/api/admin/*` | управление пользователями, событиями, категориями |
| `/api/categories/` | GET список, GET :id |

### 3.6 Сид при старте

- `services.RunSeed()` вызывается в `main.go` после `initDefaultAdmin()`.
- Если в БД уже ≥5 событий — сид пропускается.
- Создаёт: категории, 10 мероприятий, 10 тестовых пользователей, участников, отзывы.
- Изображения: `/uploads/events/event-rock.jpg`, `event-football.jpg` и т.д. (10 файлов).

### 3.7 Dockerfile

- Multi-stage: `golang:1.24-alpine` → `alpine:latest`.
- Копирует `uploads/events/` в образ.
- CMD: `./bekend`.

---

## 4. Frontend (frontend-MyAfisha)

### 4.1 Структура

```
frontend-MyAfisha/
├── package.json
├── vite.config.ts
├── tailwind.config.js
├── tsconfig.json
├── Dockerfile
├── nginx.conf
├── src/
│   ├── app/           # router, layout, pages
│   ├── modules/      # auth, events, user, admin
│   └── shared/       # api, ui, lib, constants
```

### 4.2 Зависимости

- React 18, React Router DOM
- TanStack React Query, Axios
- React Hook Form, Zod
- Tailwind CSS, Radix UI, Lucide React

### 4.3 Переменные окружения

```
VITE_BASE_API_URL=http://localhost:8081/api
VITE_YANDEX_API_KEY=   # для карт (опционально)
```

### 4.4 Основные страницы

- `/` — список событий (вкладки: Мои, Активные, Прошедшие)
- `/events/:id` — детали события
- `/profile` — профиль пользователя
- `/tickets` — мои афиши
- `/admin` — админ-панель (пользователи, события, категории)
- `/signin`, `/signup` — авторизация

### 4.5 Фичи на главной

- **HotEventsStrip** — полоса «Горящие события» (API: `GET /api/events/hot`)
- **RecommendedEvents** — «Рекомендовано для вас» (API: `GET /api/events/recommended`, auth)

### 4.6 Dockerfile

- Build: Node 20, `npm ci`, `npm run build` (Vite)
- Runtime: nginx:alpine, статика из `dist/`.

---

## 5. Запуск проекта

### 5.1 Через Docker Compose (корень ls/)

```bash
cd ls
docker-compose up -d --build
```

Сервисы:
- `postgres` — порт 5433
- `app` — порт 8081
- `frontend` — порт 5173

### 5.2 Локально (без Docker)

**Backend:**
```bash
cd back_xaxaton2025
# PostgreSQL должен быть запущен на localhost:5432 (или 5433)
cp env.template .env
# Заполнить JWT_SECRET, DB_*
go run .
```

**Frontend:**
```bash
cd frontend-MyAfisha
npm install
echo "VITE_BASE_API_URL=http://localhost:8081/api" > .env
npm run dev
```

### 5.3 Тестовые пользователи

| Email | Пароль | Роль |
|-------|--------|------|
| admin@system.local | Admin123! | Администратор |
| user1@test.local | User123! | Пользователь |
| ivan.petrov@test.local | Test123! | Пользователь |

---

## 6. Чеклист воспроизведения

### Backend
- [ ] Клонировать `back_xaxaton2025`
- [ ] Установить Go 1.24+
- [ ] Создать `.env` из `env.template`
- [ ] Запустить PostgreSQL
- [ ] `go mod download && go run .`
- [ ] Проверить `http://localhost:8081/health`
- [ ] Проверить `http://localhost:8081/swagger/index.html`

### Frontend
- [ ] Клонировать `frontend-MyAfisha`
- [ ] Установить Node 20+
- [ ] `npm install`
- [ ] Создать `.env` с `VITE_BASE_API_URL`
- [ ] `npm run dev`
- [ ] Проверить `http://localhost:5173`

### Docker
- [ ] Создать корневой `docker-compose.yml` (postgres, app, frontend)
- [ ] Убедиться, что `uploads/events/` содержит 10 изображений
- [ ] `docker-compose up -d --build`

---

## 7. Репозитории

- Backend: `https://github.com/IT-TECH-Hackaton/back_xaxaton2025`
- Frontend: `https://github.com/IT-TECH-Hackaton/frontend-MyAfisha`

---

## 8. Дополнительные ТЗ

- `ТЗ_ФИЧИ.md` — детальное описание фич «Горящие события» и «Рекомендации».
