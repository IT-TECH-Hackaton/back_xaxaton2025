# Bekend Backend - Система электронной афиши

Бекенд для системы электронной афиши на Go с полной функциональностью авторизации, управления событиями и администрирования.

## Технологии

- Go 1.23
- Gin - веб-фреймворк
- GORM - ORM для работы с БД
- PostgreSQL 15 - база данных
- JWT - аутентификация
- bcrypt - хеширование паролей
- gomail - отправка email (Yandex)
- Docker + docker-compose - контейнеризация

## Быстрый старт

### 1. Настройка .env файла

Создайте `.env` файл из `env.template`:

```bash
cp env.template .env
```

Заполните обязательные поля:
```env
EMAIL_USER=your-email@yandex.ru
EMAIL_PASSWORD=your-password
EMAIL_FROM=your-email@yandex.ru
JWT_SECRET=your-secret-key-min-32-characters-long
```

**Важно:** Включите IMAP в настройках Yandex почты:
- https://mail.yandex.ru → Настройки → Почтовые программы → IMAP

### 2. Запуск через Docker

```bash
docker-compose up -d
```

### 3. Создание администратора

```bash
docker-compose exec app go run scripts/create_admin.go admin@example.com Admin123! "Иванов Иван Иванович"
```

### 4. Проверка работы

- Health check: http://localhost:8081/health
- API: http://localhost:8081/api

## API Endpoints

### Авторизация
- `POST /api/auth/register` - Регистрация
- `POST /api/auth/verify-email` - Подтверждение email
- `POST /api/auth/resend-code` - Повторная отправка кода
- `POST /api/auth/login` - Вход
- `POST /api/auth/logout` - Выход
- `POST /api/auth/forgot-password` - Запрос восстановления пароля
- `POST /api/auth/reset-password` - Сброс пароля

### Пользователь (требуется токен)
- `GET /api/user/profile` - Получить профиль
- `PUT /api/user/profile` - Обновить профиль

### Загрузка файлов (требуется токен)
- `POST /api/upload/image` - Загрузить изображение

### События
- `GET /api/events?tab=active|my|past` - Список событий
- `GET /api/events/:id` - Детали события
- `POST /api/events` - Создать событие (требуется токен)
- `PUT /api/events/:id` - Обновить событие (требуется токен)
- `DELETE /api/events/:id` - Удалить событие (требуется токен)
- `POST /api/events/:id/join` - Подтвердить участие (требуется токен)
- `DELETE /api/events/:id/leave` - Отменить участие (требуется токен)
- `GET /api/events/:id/export` - Экспорт участников XLSX (требуется токен)

### Администрирование (требуется токен администратора)
- `GET /api/admin/users` - Список пользователей
- `GET /api/admin/users/:id` - Детали пользователя
- `PUT /api/admin/users/:id` - Обновить пользователя
- `POST /api/admin/users/:id/reset-password` - Сбросить пароль пользователя
- `DELETE /api/admin/users/:id` - Удалить пользователя
- `GET /api/admin/events` - Список всех событий

## Авторизация

Все защищенные эндпоинты требуют заголовок:
```
Authorization: Bearer <token>
```

## Структура проекта

```
bekend/
├── config/          # Конфигурация приложения
├── database/        # Подключение к БД
├── models/          # Модели данных
├── handlers/        # HTTP обработчики
├── middleware/      # JWT аутентификация
├── services/        # Email и Cron сервисы
├── utils/           # Вспомогательные функции
├── routes/          # Маршрутизация
├── scripts/         # Утилиты
└── main.go          # Точка входа
```

## Особенности

- Автоматическое обновление статусов событий (cron)
- Email уведомления о событиях
- Экспорт участников в XLSX
- Валидация данных
- JWT аутентификация
- Загрузка изображений

## Docker

```bash
# Запуск
docker-compose up -d

# Логи
docker-compose logs -f app

# Остановка
docker-compose down
```

## Лицензия

Проект создан для системы электронной афиши.
