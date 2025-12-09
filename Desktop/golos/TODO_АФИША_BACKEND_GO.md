# TODO LIST: Система электронной афиши (BACKEND - Go + Gin)

## 📋 ОБЩАЯ СТРУКТУРА ПРОЕКТА

### Этап 0: Подготовка и настройка проекта
- [x] Добавить зависимости в `go.mod`:
  - [x] `github.com/gin-gonic/gin` (уже есть)
  - [x] `github.com/go-playground/validator/v10` (уже есть)
  - [x] `gorm.io/gorm` (ORM для работы с БД)
  - [x] `gorm.io/driver/postgres` (драйвер PostgreSQL)
  - [x] `github.com/golang-jwt/jwt/v5` (JWT токены)
  - [x] `golang.org/x/crypto` (bcrypt для хеширования паролей, уже есть)
  - [x] `github.com/joho/godotenv` (загрузка .env файлов)
  - [x] `gopkg.in/gomail.v2` или `github.com/go-mail/mail` (отправка email)
  - [x] `github.com/google/uuid` (генерация UUID)
  - [ ] `golang.org/x/time/rate` (rate limiting, опционально)
- [x] Создать структуру папок:
  - [x] `internal/models/` (модели данных)
  - [x] `internal/repository/` (слой доступа к данным)
  - [x] `internal/service/` (бизнес-логика)
  - [x] `internal/dto/` (Data Transfer Objects)
  - [x] `internal/middleware/` (middleware для Gin)
  - [x] `internal/database/` (настройка БД)
  - [x] `internal/utils/` (утилиты)
- [x] Настроить PostgreSQL в `docker-compose.yml`
- [x] Создать файл миграций или настроить GORM AutoMigrate
- [x] Настроить переменные окружения в `.env.example`
- [x] Расширить `internal/config/config.go` для новых настроек

---

## 🔐 МОДУЛЬ 1: АВТОРИЗАЦИЯ (BACKEND - Go)

### 1.1. Модели данных (internal/models/)
- [x] Создать `internal/models/user.go`:
  - [x] Структура `User` (ID, Email, PasswordHash, FullName, Role, Status, CreatedAt, UpdatedAt)
  - [x] Теги GORM для полей
  - [x] Метод ComparePassword для проверки пароля
- [x] Создать `internal/models/email_verification.go`:
  - [x] Структура `EmailVerification` (ID, Email, Code, PasswordHash, FullName, ExpiresAt, CreatedAt)
  - [x] Теги GORM
  - [x] Метод IsExpired()
- [x] Создать `internal/models/password_reset.go`:
  - [x] Структура `PasswordReset` (ID, UserID, Token, ExpiresAt, CreatedAt)
  - [x] Теги GORM, связь с User
  - [x] Метод IsExpired()
- [x] Создать `internal/models/enums.go`:
  - [x] Тип `UserRole` (USER, ADMIN)
  - [x] Тип `UserStatus` (ACTIVE, DELETED)
  - [x] Тип `EventStatus` (ACTIVE, PAST, REJECTED)
  - [x] Методы String() для типов

### 1.2. Настройка базы данных
- [x] Создать `internal/database/database.go`:
  - [x] Функция `Connect()` для подключения к PostgreSQL
  - [x] Настройка connection pool
  - [x] Функция `Migrate()` для автоматических миграций
  - [x] Функция `Close()` для закрытия соединения
- [x] Добавить настройки БД в `internal/config/config.go`:
  - [x] `DatabaseConfig` (Host, Port, User, Password, DBName, SSLMode)
- [x] Добавить PostgreSQL сервис в `docker-compose.yml`
- [x] Обновить `cmd/api/main.go` для инициализации БД

### 1.3. Repository слой (internal/repository/)
- [x] Создать `internal/repository/user_repository.go`:
  - [x] Структура `UserRepository` с полем `*gorm.DB`
  - [x] Метод `Create(user *models.User) error`
  - [x] Метод `FindByEmail(email string) (*models.User, error)`
  - [x] Метод `FindByID(id uint) (*models.User, error)`
  - [x] Метод `Update(user *models.User) error`
  - [x] Метод `Delete(id uint) error` (soft delete)
  - [x] Метод `EmailExists(email string) bool`
  - [ ] Метод `FindAll(filters) ([]models.User, int64, error)` с пагинацией (для админа)
- [x] Создать `internal/repository/email_verification_repository.go`:
  - [x] Метод `Create(ev *models.EmailVerification) error`
  - [x] Метод `FindByEmailAndCode(email, code string) (*models.EmailVerification, error)`
  - [x] Метод `DeleteByEmail(email string) error`
- [x] Создать `internal/repository/password_reset_repository.go`:
  - [x] Метод `Create(pr *models.PasswordReset) error`
  - [x] Метод `FindByToken(token string) (*models.PasswordReset, error)`
  - [x] Метод `Delete(id uint) error`

### 1.4. DTO (internal/dto/)
- [x] Создать `internal/dto/auth.go`:
  - [x] `LoginRequest` (Email, Password) с валидацией
  - [x] `LoginResponse` (Token, RefreshToken, User)
  - [x] `RegisterRequest` (FullName, Email, Password, ConfirmPassword) с валидацией
  - [x] `VerifyEmailRequest` (Email, Code) с валидацией
  - [x] `ResendVerificationCodeRequest` (Email) с валидацией
  - [x] `ForgotPasswordRequest` (Email) с валидацией
  - [x] `ResetPasswordRequest` (Token, NewPassword, ConfirmPassword) с валидацией
  - [x] `RefreshTokenRequest` (RefreshToken)
  - [x] `RefreshTokenResponse` (Token, RefreshToken)
  - [x] `UserResponse` (ID, Email, FullName, Role, Status, CreatedAt)

### 1.5. Service слой (internal/service/auth/)
- [ ] Создать `internal/service/auth/auth_service.go`:
  - [ ] Структура `AuthService` с зависимостями (repositories, email service, jwt service)
  - [ ] Метод `Login(req dto.LoginRequest) (*dto.LoginResponse, error)`
  - [ ] Метод `Register(req dto.RegisterRequest) error`
  - [ ] Метод `VerifyEmail(req dto.VerifyEmailRequest) (*dto.LoginResponse, error)`
  - [ ] Метод `ResendVerificationCode(email string) error`
  - [ ] Метод `ForgotPassword(req dto.ForgotPasswordRequest) error`
  - [ ] Метод `ResetPassword(req dto.ResetPasswordRequest) error`
  - [ ] Метод `RefreshToken(req dto.RefreshTokenRequest) (*dto.RefreshTokenResponse, error)`
- [ ] Создать `internal/service/auth/password_service.go`:
  - [ ] Метод `HashPassword(password string) (string, error)` (bcrypt)
  - [ ] Метод `ComparePassword(hashedPassword, password string) bool`
  - [ ] Метод `ValidatePassword(password string) error` (латиница, цифры, символы, мин. 8)
- [ ] Создать `internal/service/auth/jwt_service.go`:
  - [ ] Структура `JWTService` с секретным ключом
  - [ ] Метод `GenerateToken(userID uint, role string) (string, error)` (access token, 15 мин)
  - [ ] Метод `GenerateRefreshToken(userID uint) (string, error)` (refresh token, 7 дней)
  - [ ] Метод `ValidateToken(tokenString string) (*Claims, error)`
  - [ ] Структура `Claims` (UserID, Role, ExpiresAt)

### 1.6. Handlers (internal/api/handlers/auth_handler.go)
- [x] Создать структуру `AuthHandler` с зависимостью `*service.AuthService`
- [x] Метод `Login(c *gin.Context)`:
  - [x] Биндинг `dto.LoginRequest` с валидацией
  - [x] Вызов `authService.Login()`
  - [x] Возврат `dto.LoginResponse` или ошибки
- [x] Метод `Register(c *gin.Context)`:
  - [x] Биндинг `dto.RegisterRequest` с валидацией
  - [x] Вызов `authService.Register()`
  - [x] Возврат успешного ответа
- [x] Метод `VerifyEmail(c *gin.Context)`:
  - [x] Биндинг `dto.VerifyEmailRequest`
  - [x] Вызов `authService.VerifyEmail()`
  - [x] Возврат токенов
- [x] Метод `ResendVerificationCode(c *gin.Context)`
- [x] Метод `ForgotPassword(c *gin.Context)`
- [x] Метод `ResetPassword(c *gin.Context)`
- [x] Метод `RefreshToken(c *gin.Context)`

### 1.7. Middleware (internal/middleware/)
- [x] Создать `internal/middleware/auth_middleware.go`:
  - [x] Функция `AuthMiddleware(jwtService *service.JWTService) gin.HandlerFunc`
  - [x] Извлечение токена из заголовка `Authorization: Bearer <token>`
  - [x] Валидация токена через `jwtService.ValidateToken()`
  - [x] Добавление данных пользователя в `c.Set("userID", userID)` и `c.Set("userRole", role)`
  - [x] Обработка ошибок (401 Unauthorized)
- [x] Создать `internal/middleware/admin_middleware.go`:
  - [x] Функция `AdminMiddleware() gin.HandlerFunc`
  - [x] Проверка роли из контекста (`c.Get("userRole")`)
  - [x] Возврат 403 Forbidden если не админ

### 1.8. Router (internal/api/router.go)
- [x] Добавить группу `/api/v1/auth`:
  - [x] `POST /api/v1/auth/login` -> `authHandler.Login`
  - [x] `POST /api/v1/auth/register` -> `authHandler.Register`
  - [x] `POST /api/v1/auth/verify-email` -> `authHandler.VerifyEmail`
  - [x] `POST /api/v1/auth/resend-verification-code` -> `authHandler.ResendVerificationCode`
  - [x] `POST /api/v1/auth/forgot-password` -> `authHandler.ForgotPassword`
  - [x] `POST /api/v1/auth/reset-password` -> `authHandler.ResetPassword`
  - [x] `POST /api/v1/auth/refresh` -> `authHandler.RefreshToken`

### 1.9. Email сервис (internal/service/email/)
- [x] Создать `internal/service/email/email_service.go`:
  - [x] Структура `EmailService` с настройками SMTP
  - [x] Метод `SendVerificationCode(email, code string) error`
  - [x] Метод `SendWelcomeEmail(email, fullName string) error`
  - [x] Метод `SendPasswordResetLink(email, token string) error`
  - [x] Метод `SendPasswordChangedNotification(email string) error`
  - [x] Метод `SendEventNotification(email, subject, body string) error` (для событий)
- [ ] Создать `internal/service/email/templates.go`:
  - [ ] HTML шаблоны для всех типов писем (опционально, сейчас встроены в методы)
  - [ ] Единый стиль оформления (реализован в методах)
- [x] Добавить настройки SMTP в `internal/config/config.go`:
  - [x] `EmailConfig` (Host, Port, User, Password, From)

### 1.10. Валидация
- [x] Использовать `github.com/go-playground/validator/v10` для валидации DTO
- [x] Создать кастомные валидаторы в `internal/utils/validators.go`:
  - [x] `ValidateEmail(email string) bool` (regex)
  - [x] `ValidatePassword(password string) bool` (латиница, цифры, символы, мин. 8)
  - [x] `ValidateFullName(name string) bool` (только русские буквы)
- [x] Добавить теги валидации в DTO структуры
- [x] Создать функцию для обработки ошибок валидации в handlers (используется ShouldBindJSON)

---

## 🎉 МОДУЛЬ 2: СОБЫТИЯ (BACKEND - Go)

### 2.1. Модели данных (internal/models/)
- [ ] Создать `internal/models/event.go`:
  - [ ] Структура `Event` (ID, Title, ShortDescription, FullDescription, StartDate, EndDate, ImageURL, PaymentInfo, MaxParticipants, Status, OrganizerID, CreatedAt, UpdatedAt)
  - [ ] Связь `Organizer *User` (belongs to)
  - [ ] Связь `Participants []EventParticipant` (has many)
  - [ ] Теги GORM
  - [ ] Метод `IsActive() bool` (проверка дат)
  - [ ] Метод `IsPast() bool`
- [ ] Создать `internal/models/event_participant.go`:
  - [ ] Структура `EventParticipant` (ID, EventID, UserID, ConfirmedAt, CreatedAt)
  - [ ] Связи `Event *Event`, `User *User`
  - [ ] Уникальный индекс на (EventID, UserID)
- [ ] Создать `internal/models/event_rating.go` (опционально):
  - [ ] Структура `EventRating` (ID, EventID, UserID, Rating, Comment, CreatedAt)
  - [ ] Связи с Event и User
- [ ] Создать `internal/models/enums.go` (дополнить):
  - [ ] Тип `EventStatus` (ACTIVE, PAST, REJECTED)

### 2.2. Repository слой (internal/repository/)
- [ ] Создать `internal/repository/event_repository.go`:
  - [ ] Метод `Create(event *models.Event) error`
  - [ ] Метод `FindByID(id uint) (*models.Event, error)` с preload участников
  - [ ] Метод `FindAll(filters) ([]models.Event, int64, error)`:
    - [ ] Фильтрация по статусу
    - [ ] Исключение REJECTED для обычных пользователей
    - [ ] Пагинация
    - [ ] Сортировка по дате начала
  - [ ] Метод `FindByUserID(userID uint) ([]models.Event, error)` (мои события)
  - [ ] Метод `Update(event *models.Event) error`
  - [ ] Метод `Delete(id uint) error`
  - [ ] Метод `UpdateStatus(eventID uint, status models.EventStatus) error`
- [ ] Создать `internal/repository/event_participant_repository.go`:
  - [ ] Метод `Create(ep *models.EventParticipant) error`
  - [ ] Метод `FindByEventAndUser(eventID, userID uint) (*models.EventParticipant, error)`
  - [ ] Метод `Delete(eventID, userID uint) error`
  - [ ] Метод `CountByEventID(eventID uint) (int64, error)`
  - [ ] Метод `FindByEventID(eventID uint) ([]models.EventParticipant, error)`

### 2.3. DTO (internal/dto/)
- [ ] Создать `internal/dto/event.go`:
  - [ ] `CreateEventRequest` (все поля события) с валидацией
  - [ ] `UpdateEventRequest` (все поля опциональные) с валидацией
  - [ ] `GetEventsRequest` (Status, Page, Limit) с валидацией
  - [ ] `EventResponse` (все поля + количество участников + статус участия)
  - [ ] `EventsListResponse` (Events []EventResponse, Total, Page, Limit)
  - [ ] `ParticipateRequest` (EventID)
  - [ ] `ExportParticipantsRequest` (EventID, Format)

### 2.4. Service слой (internal/service/event/)
- [ ] Создать `internal/service/event/event_service.go`:
  - [ ] Структура `EventService` с зависимостями
  - [ ] Метод `CreateEvent(req dto.CreateEventRequest, organizerID uint) (*models.Event, error)`
  - [ ] Метод `GetEvents(req dto.GetEventsRequest, userID uint) (*dto.EventsListResponse, error)`
  - [ ] Метод `GetMyEvents(userID uint, page, limit int) (*dto.EventsListResponse, error)`
  - [ ] Метод `GetEventByID(eventID, userID uint) (*dto.EventResponse, error)`
  - [ ] Метод `UpdateEvent(eventID uint, req dto.UpdateEventRequest) (*models.Event, error)`
  - [ ] Метод `DeleteEvent(eventID uint) error`
  - [ ] Метод `Participate(eventID, userID uint) error`
  - [ ] Метод `CancelParticipation(eventID, userID uint) error`
  - [ ] Метод `UpdateEventStatuses() error` (для cron job)
- [ ] Создать `internal/service/event/file_service.go`:
  - [ ] Метод `SaveImage(file multipart.File, header *multipart.FileHeader) (string, error)`
  - [ ] Метод `DeleteImage(imageURL string) error`
  - [ ] Валидация типа и размера файла

### 2.5. Handlers (internal/api/handlers/event_handler.go)
- [ ] Создать структуру `EventHandler` с зависимостью `*service.EventService`
- [ ] Метод `GetEvents(c *gin.Context)`:
  - [ ] Биндинг query параметров
  - [ ] Получение userID из контекста (если авторизован)
  - [ ] Вызов `eventService.GetEvents()`
  - [ ] Возврат списка событий
- [ ] Метод `GetMyEvents(c *gin.Context)`:
  - [ ] Получение userID из контекста (обязательно)
  - [ ] Вызов `eventService.GetMyEvents()`
- [ ] Метод `GetEventByID(c *gin.Context)`:
  - [ ] Парсинг ID из URL
  - [ ] Получение userID из контекста (опционально)
  - [ ] Вызов `eventService.GetEventByID()`
- [ ] Метод `Participate(c *gin.Context)`
- [ ] Метод `CancelParticipation(c *gin.Context)`

### 2.6. Router (internal/api/router.go)
- [ ] Добавить группу `/api/v1/events`:
  - [ ] `GET /api/v1/events` -> `eventHandler.GetEvents` (публичный)
  - [ ] `GET /api/v1/events/my-events` -> `eventHandler.GetMyEvents` (требует auth)
  - [ ] `GET /api/v1/events/:id` -> `eventHandler.GetEventByID` (публичный)
  - [ ] `POST /api/v1/events/:id/participate` -> `eventHandler.Participate` (требует auth)
  - [ ] `DELETE /api/v1/events/:id/participate` -> `eventHandler.CancelParticipation` (требует auth)

### 2.7. Автоматическое обновление статусов
- [ ] Создать `internal/service/event/status_updater.go`:
  - [ ] Метод `UpdateEventStatuses() error`
  - [ ] Логика определения статуса по датам
  - [ ] Обновление статусов в БД
- [ ] Создать `internal/cron/cron.go`:
  - [ ] Настройка cron job (например, `github.com/robfig/cron/v3`)
  - [ ] Запуск `UpdateEventStatuses()` каждый час
  - [ ] Интеграция в `cmd/api/main.go`

---

## 👨‍💼 МОДУЛЬ 3: АДМИНИСТРИРОВАНИЕ (BACKEND - Go)

### 3.1. Repository слой (расширение)
- [ ] Расширить `internal/repository/user_repository.go`:
  - [ ] Метод `FindAllWithFilters(filters dto.GetUsersRequest) ([]models.User, int64, error)`:
    - [ ] Фильтрация по ФИО (LIKE)
    - [ ] Фильтрация по ролям (IN)
    - [ ] Фильтрация по статусу
    - [ ] Фильтрация по диапазону дат
    - [ ] Пагинация

### 3.2. DTO (internal/dto/)
- [ ] Создать `internal/dto/admin.go`:
  - [ ] `GetUsersRequest` (FullName, Roles[], Status, DateFrom, DateTo, Page, Limit)
  - [ ] `UpdateUserRequest` (FullName, Role)
  - [ ] `AdminResetPasswordRequest` (UserID, NewPassword)
  - [ ] `UserResponse` (все поля без пароля)
  - [ ] `UsersListResponse` (Users []UserResponse, Total, Page, Limit)
  - [ ] `GetAdminEventsRequest` (Status, Page, Limit)
  - [ ] `RejectEventRequest` (EventID)

### 3.3. Service слой (internal/service/admin/)
- [ ] Создать `internal/service/admin/user_service.go`:
  - [ ] Структура `AdminUserService`
  - [ ] Метод `GetUsers(req dto.GetUsersRequest) (*dto.UsersListResponse, error)`
  - [ ] Метод `GetUserByID(userID uint) (*dto.UserResponse, error)`
  - [ ] Метод `UpdateUser(userID uint, req dto.UpdateUserRequest) (*dto.UserResponse, error)`
  - [ ] Метод `ResetPassword(userID uint, newPassword string) error`
  - [ ] Метод `DeleteUser(userID uint) error` (soft delete)
- [ ] Создать `internal/service/admin/event_service.go`:
  - [ ] Структура `AdminEventService` (расширение обычного EventService)
  - [ ] Метод `GetAllEvents(req dto.GetAdminEventsRequest) (*dto.EventsListResponse, error)` (включая REJECTED)
  - [ ] Метод `CreateEvent(req dto.CreateEventRequest, organizerID uint) (*models.Event, error)`
  - [ ] Метод `UpdateEvent(eventID uint, req dto.UpdateEventRequest) (*models.Event, error)`
  - [ ] Метод `RejectEvent(eventID uint) error`
  - [ ] Метод `DeleteEvent(eventID uint) error`
  - [ ] Метод `ExportParticipants(eventID uint, format string) ([]byte, string, error)`

### 3.4. Handlers (internal/api/handlers/admin_handler.go)
- [ ] Создать структуру `AdminHandler` с зависимостями
- [ ] Метод `GetUsers(c *gin.Context)`
- [ ] Метод `GetUserByID(c *gin.Context)`
- [ ] Метод `UpdateUser(c *gin.Context)`
- [ ] Метод `ResetPassword(c *gin.Context)`
- [ ] Метод `DeleteUser(c *gin.Context)`
- [ ] Метод `GetAllEvents(c *gin.Context)`
- [ ] Метод `CreateEvent(c *gin.Context)`
- [ ] Метод `UpdateEvent(c *gin.Context)`
- [ ] Метод `RejectEvent(c *gin.Context)`
- [ ] Метод `DeleteEvent(c *gin.Context)`
- [ ] Метод `ExportParticipants(c *gin.Context)`

### 3.5. Router (internal/api/router.go)
- [ ] Добавить группу `/api/v1/admin` с `AdminMiddleware()`:
  - [ ] `/api/v1/admin/users`:
    - [ ] `GET /api/v1/admin/users` -> `adminHandler.GetUsers`
    - [ ] `GET /api/v1/admin/users/:id` -> `adminHandler.GetUserByID`
    - [ ] `PUT /api/v1/admin/users/:id` -> `adminHandler.UpdateUser`
    - [ ] `POST /api/v1/admin/users/:id/reset-password` -> `adminHandler.ResetPassword`
    - [ ] `DELETE /api/v1/admin/users/:id` -> `adminHandler.DeleteUser`
  - [ ] `/api/v1/admin/events`:
    - [ ] `GET /api/v1/admin/events` -> `adminHandler.GetAllEvents`
    - [ ] `POST /api/v1/admin/events` -> `adminHandler.CreateEvent`
    - [ ] `PUT /api/v1/admin/events/:id` -> `adminHandler.UpdateEvent`
    - [ ] `POST /api/v1/admin/events/:id/reject` -> `adminHandler.RejectEvent`
    - [ ] `DELETE /api/v1/admin/events/:id` -> `adminHandler.DeleteEvent`
    - [ ] `GET /api/v1/admin/events/:id/export-participants` -> `adminHandler.ExportParticipants`

---

## 📧 ИНТЕГРАЦИЯ С ПОЧТОВЫМ СЕРВИСОМ (BACKEND - Go)

### 4.1. Email сервис (расширение)
- [ ] Расширить `internal/service/email/email_service.go`:
  - [ ] Метод `SendEventCreatedNotification(event *models.Event, participants []models.User) error`
  - [ ] Метод `SendEventUpdatedNotification(event *models.Event, participants []models.User) error`
  - [ ] Метод `SendEventReminder(event *models.Event, participants []models.User) error` (24 часа до начала)
  - [ ] Метод `SendParticipationConfirmationToOrganizer(event *models.Event, participant *models.User) error`
  - [ ] Метод `SendParticipationCancellationToOrganizer(event *models.Event, participant *models.User) error`

### 4.2. Фоновые задачи
- [ ] Создать `internal/service/event/reminder_service.go`:
  - [ ] Метод `SendEventReminders() error`
  - [ ] Поиск событий, которые начинаются через 24 часа
  - [ ] Отправка уведомлений всем участникам
  - [ ] Предотвращение дублирования (флаг в БД или кеш)
- [ ] Добавить cron job для отправки напоминаний (каждый час)

---

## 🎨 ДОПОЛНИТЕЛЬНЫЙ ФУНКЦИОНАЛ (BACKEND - Go)

### 5.1. Экспорт списка участников
- [ ] Создать `internal/service/admin/export_service.go`:
  - [ ] Метод `ExportParticipantsToCSV(participants []models.EventParticipant) ([]byte, error)`
  - [ ] Метод `ExportParticipantsToXLSX(participants []models.EventParticipant) ([]byte, error)`
  - [ ] Использовать библиотеку `github.com/xuri/excelize/v2` для XLSX
  - [ ] Генерация CSV вручную или через библиотеку
- [ ] Добавить handler `ExportParticipants` в `admin_handler.go`

### 5.2. Рейтинг/отзывы о событиях (опционально)
- [ ] Расширить `internal/repository/event_rating_repository.go`
- [ ] Создать `internal/service/event/rating_service.go`
- [ ] Создать DTO для рейтингов
- [ ] Добавить handlers и routes

---

## 🔒 БЕЗОПАСНОСТЬ И НАДЕЖНОСТЬ (BACKEND - Go)

### 6.1. Аутентификация и авторизация
- [ ] Реализовать JWT с правильными сроками действия
- [ ] Реализовать refresh token механизм
- [ ] Сохранять refresh tokens в БД (таблица `refresh_tokens`)
- [ ] Реализовать logout (удаление refresh token)
- [ ] Добавить rate limiting для auth endpoints (использовать существующий middleware)

### 6.2. Защита данных
- [ ] Использовать bcrypt для хеширования паролей (cost 10-12)
- [ ] Валидировать все входные данные через `validator/v10`
- [ ] Санитизировать данные (защита от XSS)
- [ ] Использовать GORM prepared statements (автоматически)
- [ ] Настроить CORS правильно (использовать существующий middleware)
- [ ] Добавить helmet-like middleware для безопасности заголовков

### 6.3. Валидация
- [ ] Создать кастомные валидаторы в `internal/utils/validators.go`
- [ ] Зарегистрировать валидаторы в `validator/v10`
- [ ] Использовать теги валидации во всех DTO
- [ ] Создать функцию для форматирования ошибок валидации

### 6.4. Обработка ошибок
- [ ] Создать `internal/api/errors.go` (расширить существующий):
  - [ ] Кастомные типы ошибок
  - [ ] Функция `HandleError(c *gin.Context, err error)`
  - [ ] Единообразные ответы об ошибках
- [ ] Использовать существующий `recoveryMiddleware()` для паник

### 6.5. Логирование
- [ ] Использовать существующий `internal/logger/logger.go`
- [ ] Добавить логирование во все критичные операции
- [ ] Логировать ошибки с контекстом
- [ ] Настроить уровни логирования

---

## 🎯 КАЧЕСТВО КОДА И АРХИТЕКТУРА (BACKEND - Go)

### 7.1. Структура проекта
- [ ] Следовать существующей структуре:
  - [ ] `cmd/api/main.go` - точка входа
  - [ ] `internal/api/` - handlers, router, middleware
  - [ ] `internal/service/` - бизнес-логика
  - [ ] `internal/repository/` - доступ к данным
  - [ ] `internal/models/` - модели данных
  - [ ] `internal/dto/` - DTO
  - [ ] `internal/config/` - конфигурация
  - [ ] `internal/database/` - настройка БД
  - [ ] `internal/utils/` - утилиты

### 7.2. Паттерны
- [ ] Использовать Dependency Injection (передавать зависимости через конструкторы)
- [ ] Repository pattern для доступа к данным
- [ ] Service layer для бизнес-логики
- [ ] DTO для передачи данных между слоями

### 7.3. База данных
- [ ] Использовать GORM для работы с БД
- [ ] Настроить миграции через GORM AutoMigrate
- [ ] Создать индексы для оптимизации:
  - [ ] `users.email` (уникальный)
  - [ ] `events.status`, `events.start_date`, `events.end_date`
  - [ ] `event_participants.event_id`, `event_participants.user_id`
  - [ ] Составной индекс на `(event_id, user_id)` для EventParticipant
- [ ] Настроить связи (foreign keys) в GORM

### 7.4. Тестирование
- [ ] Написать unit тесты для сервисов (использовать `testing` пакет)
- [ ] Написать unit тесты для репозиториев (с тестовой БД)
- [ ] Написать integration тесты для API (использовать `httptest`)
- [ ] Использовать моки для зависимостей (например, `github.com/stretchr/testify/mock`)

---

## 🚀 ПРОИЗВОДИТЕЛЬНОСТЬ И ОПТИМИЗАЦИЯ (BACKEND - Go)

### 8.1. Оптимизация БД
- [ ] Создать индексы на часто используемые поля
- [ ] Использовать `Preload()` для eager loading связанных данных
- [ ] Реализовать пагинацию для всех списков
- [ ] Оптимизировать запросы (избегать N+1 проблем)
- [ ] Настроить connection pool в GORM

### 8.2. Кеширование (опционально)
- [ ] Добавить Redis в `docker-compose.yml`
- [ ] Создать `internal/service/cache/cache_service.go`
- [ ] Кешировать список активных событий
- [ ] Кешировать данные пользователя
- [ ] Инвалидировать кеш при обновлениях

---

## 📝 ДОКУМЕНТАЦИЯ И API (BACKEND - Go)

### 9.1. API Документация
- [ ] Настроить Swagger для Gin (например, `github.com/swaggo/gin-swagger`)
- [ ] Добавить Swagger аннотации ко всем handlers
- [ ] Документировать все DTO
- [ ] Добавить примеры запросов и ответов
- [ ] Настроить авторизацию в Swagger

### 9.2. Техническая документация
- [ ] Обновить README.md с описанием нового функционала
- [ ] Документировать структуру БД
- [ ] Документировать API endpoints
- [ ] Документировать переменные окружения

---

## 🔧 ИНФРАСТРУКТУРА И ДЕПЛОЙ (BACKEND - Go)

### 10.1. Конфигурация
- [ ] Расширить `internal/config/config.go`:
  - [ ] `DatabaseConfig`
  - [ ] `EmailConfig`
  - [ ] `JWTConfig` (Secret, AccessTokenTTL, RefreshTokenTTL)
- [ ] Обновить `.env.example` с новыми переменными
- [ ] Добавить загрузку .env через `github.com/joho/godotenv`

### 10.2. Docker
- [ ] Добавить PostgreSQL сервис в `docker-compose.yml`
- [ ] Добавить Redis сервис (опционально, для кеширования)
- [ ] Обновить `Dockerfile` если необходимо
- [ ] Настроить health checks для всех сервисов
- [ ] Настроить volumes для PostgreSQL данных

### 10.3. Миграции
- [ ] Создать функцию миграций в `internal/database/migrate.go`
- [ ] Вызывать миграции при старте приложения
- [ ] Или использовать отдельную команду для миграций

---

## ✅ ПРИОРИТЕТЫ ВЫПОЛНЕНИЯ

### Высокий приоритет (MVP):
1. Настройка БД и моделей
2. Модуль авторизации (полностью)
3. Базовый функционал модуля событий
4. Базовый функционал администрирования
5. Интеграция с почтовым сервисом
6. Безопасность и валидация

### Средний приоритет:
1. Дополнительный функционал (экспорт, рейтинги)
2. Оптимизация производительности
3. Кеширование
4. Автоматическое обновление статусов событий

### Низкий приоритет:
1. Расширенное тестирование
2. Дополнительные фичи
3. Расширенная документация

---

## 📊 ОЦЕНКА ВРЕМЕНИ (примерная)

- **Настройка БД и инфраструктуры**: 8-12 часов
- **Модуль Авторизация**: 30-40 часов
- **Модуль События**: 40-50 часов
- **Модуль Администрирование**: 35-45 часов
- **Интеграция с почтой**: 10-15 часов
- **Дополнительный функционал**: 15-20 часов
- **Безопасность и тестирование**: 30-40 часов
- **Оптимизация и деплой**: 15-25 часов

**Общая оценка**: 183-247 часов (23-31 рабочий день)

---

## 🛠️ СТЕК ТЕХНОЛОГИЙ (ПОДТВЕРЖДЕН)

### Основной стек:
- **Go 1.23** - язык программирования
- **Gin** - веб-фреймворк
- **GORM** - ORM для работы с БД
- **PostgreSQL** - база данных
- **JWT (golang-jwt/jwt/v5)** - аутентификация
- **bcrypt (golang.org/x/crypto)** - хеширование паролей
- **validator/v10** - валидация данных
- **gomail** - отправка email
- **Docker + docker-compose** - контейнеризация

### Дополнительные библиотеки:
- **github.com/google/uuid** - генерация UUID
- **github.com/robfig/cron/v3** - cron jobs
- **github.com/xuri/excelize/v2** - экспорт в XLSX
- **github.com/swaggo/gin-swagger** - Swagger документация
- **github.com/joho/godotenv** - загрузка .env

---

**Примечание**: Этот TODO list адаптирован под существующий стек проекта (Go + Gin). Все задачи учитывают текущую структуру проекта и существующие паттерны.
