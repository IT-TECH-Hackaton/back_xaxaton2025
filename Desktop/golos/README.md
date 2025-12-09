# Голосовой помощник на Go + Docker + FastAPI с GigaChat

## Архитектура проекта

### Технологический стек
- **Go** - основной бекенд сервис для обработки запросов и интеграции с GigaChat
- **FastAPI (Python)** - микросервис для обработки аудио (STT/TTS)
- **Docker** - контейнеризация всех сервисов
- **GigaChat API** - обработка естественного языка и генерация ответов

### Компоненты системы

```
┌─────────────┐
│   Клиент    │
│  (Web/Mobile)│
└──────┬──────┘
       │ HTTP/WebSocket
       ▼
┌─────────────────────────────────────┐
│         API Gateway (Go)            │
│  - Маршрутизация запросов           │
│  - Аутентификация                   │
│  - Управление сессиями              │
└──────┬──────────────────┬───────────┘
       │                  │
       ▼                  ▼
┌──────────────┐   ┌──────────────┐
│ Audio Service│   │ Chat Service │
│  (FastAPI)   │   │     (Go)     │
│              │   │              │
│ - STT        │   │ - GigaChat   │
│ - TTS        │   │   интеграция │
│ - Обработка  │   │ - Контекст   │
│   аудио      │   │   диалога    │
└──────────────┘   └──────────────┘
```

## Структура проекта

```
golos/
├── cmd/
│   └── api/
│       └── main.go                 # Точка входа Go сервиса
├── internal/
│   ├── api/
│   │   ├── handlers/              # HTTP обработчики
│   │   │   ├── audio.go           # Обработка аудио запросов
│   │   │   ├── chat.go            # Обработка чат запросов
│   │   │   └── health.go          # Health check
│   │   ├── middleware/            # Middleware
│   │   │   ├── auth.go            # Аутентификация
│   │   │   └── cors.go            # CORS
│   │   └── router.go              # Маршрутизация
│   ├── service/
│   │   ├── gigachat/              # GigaChat клиент
│   │   │   ├── client.go          # HTTP клиент для GigaChat API
│   │   │   ├── auth.go            # OAuth аутентификация
│   │   │   └── models.go          # Модели данных
│   │   └── audio/                 # Интеграция с Audio Service
│   │       └── client.go          # HTTP клиент для FastAPI сервиса
│   ├── storage/
│   │   ├── session.go             # Управление сессиями
│   │   └── cache.go               # Кэширование
│   └── config/
│       └── config.go              # Конфигурация
├── audio-service/                  # Python FastAPI сервис
│   ├── app/
│   │   ├── main.py                # FastAPI приложение
│   │   ├── routers/
│   │   │   ├── stt.py             # Speech-to-Text endpoint
│   │   │   └── tts.py             # Text-to-Speech endpoint
│   │   ├── services/
│   │   │   ├── stt_service.py     # STT логика
│   │   │   └── tts_service.py     # TTS логика
│   │   └── models/
│   │       └── audio.py           # Модели данных
│   ├── requirements.txt
│   └── Dockerfile
├── docker-compose.yml              # Оркестрация сервисов
├── Dockerfile                      # Dockerfile для Go сервиса
├── go.mod
├── go.sum
├── .env.example                    # Пример переменных окружения
└── README.md
```

## План реализации

### Этап 1: Настройка инфраструктуры

1. **Инициализация Go проекта**
   - Создание `go.mod`
   - Настройка структуры проекта
   - Выбор веб-фреймворка (Gin/Fiber/Echo)

2. **Настройка Docker**
   - Dockerfile для Go сервиса
   - Dockerfile для FastAPI сервиса
   - docker-compose.yml с сервисами:
     - `api` (Go сервис)
     - `audio-service` (FastAPI)
     - `redis` (для кэширования и сессий, опционально)

3. **Конфигурация**
   - Переменные окружения для:
     - GigaChat API credentials
     - Порты сервисов
     - Таймауты
     - Секретные ключи

### Этап 2: Audio Service (FastAPI)

1. **STT (Speech-to-Text)**
   - Endpoint: `POST /api/v1/stt`
   - Принимает аудио файл (WAV, MP3, OGG)
   - Использует библиотеку для распознавания речи:
     - Вариант 1: Whisper (OpenAI) - локально или через API
     - Вариант 2: Yandex SpeechKit
     - Вариант 3: Google Speech-to-Text
   - Возвращает текст

2. **TTS (Text-to-Speech)**
   - Endpoint: `POST /api/v1/tts`
   - Принимает текст и параметры голоса
   - Генерирует аудио файл
   - Использует библиотеку для синтеза речи:
     - Вариант 1: pyttsx3 (локально)
     - Вариант 2: Yandex SpeechKit
     - Вариант 3: Google Text-to-Speech
   - Возвращает аудио файл

### Этап 3: GigaChat интеграция (Go)

1. **OAuth аутентификация**
   - Получение access token от GigaChat
   - Обновление токена при истечении
   - Кэширование токена

2. **GigaChat клиент**
   - Метод для отправки сообщений в GigaChat API
   - Обработка ответов
   - Управление контекстом диалога
   - Обработка ошибок и retry логика

3. **Модели данных**
   - Структуры для запросов/ответов GigaChat
   - Сериализация/десериализация JSON

### Этап 4: API Gateway (Go)

1. **Основные endpoints**
   - `POST /api/v1/voice/process` - полный цикл обработки голоса
   - `POST /api/v1/chat/message` - текстовый чат (опционально)
   - `GET /api/v1/health` - health check
   - `GET /api/v1/ws` - WebSocket для стриминга (опционально)

2. **Обработчик голосовых запросов**
   - Прием аудио от клиента
   - Отправка в Audio Service для STT
   - Отправка текста в GigaChat
   - Получение ответа от GigaChat
   - Отправка текста в Audio Service для TTS
   - Возврат аудио клиенту

3. **Управление сессиями**
   - Сохранение контекста диалога
   - Идентификация пользователя
   - История сообщений

### Этап 5: Дополнительные функции

1. **WebSocket поддержка**
   - Стриминг аудио в реальном времени
   - Стриминг ответов от GigaChat

2. **Кэширование**
   - Кэширование частых запросов
   - Кэширование токенов GigaChat

3. **Логирование и мониторинг**
   - Структурированное логирование
   - Метрики (опционально: Prometheus)

4. **Обработка ошибок**
   - Graceful degradation
   - Retry механизмы
   - Детальные error responses

## API Endpoints

### Go API Service

```
POST /api/v1/voice/process
Content-Type: multipart/form-data
Body: audio file

Response:
{
  "text": "распознанный текст",
  "response": "ответ от GigaChat",
  "audio": "base64 encoded audio" или URL
}
```

```
POST /api/v1/chat/message
Content-Type: application/json
Body: {
  "message": "текст сообщения",
  "session_id": "опциональный ID сессии"
}

Response: {
  "response": "ответ от GigaChat",
  "session_id": "ID сессии"
}
```

### FastAPI Audio Service

```
POST /api/v1/stt
Content-Type: multipart/form-data
Body: audio file

Response: {
  "text": "распознанный текст"
}
```

```
POST /api/v1/tts
Content-Type: application/json
Body: {
  "text": "текст для синтеза",
  "voice": "опциональный параметр голоса"
}

Response: audio file (binary)
```

## Переменные окружения

```env
# GigaChat
GIGACHAT_CLIENT_ID=your_client_id
GIGACHAT_AUTHORIZATION_KEY=your_authorization_key_base64
GIGACHAT_SCOPE=GIGACHAT_API_PERS

# API Service
API_PORT=8080
API_HOST=0.0.0.0

# Audio Service
AUDIO_SERVICE_URL=http://localhost:8000

# Session TTL (опционально, по умолчанию 30m)
SESSION_TTL=30m
```

**Важно:** Используется модель **GigaChat-Pro** для всех запросов.

## Быстрый старт

### 1. Настройка переменных окружения

Создайте файл `.env` на основе `.env.example`:

```bash
cp .env.example .env
```

Заполните `GIGACHAT_CLIENT_ID` и `GIGACHAT_CLIENT_SECRET` вашими данными от GigaChat.

### 2. Запуск через Docker Compose

```bash
# Запуск всех сервисов
docker-compose up -d

# Просмотр логов
docker-compose logs -f

# Остановка
docker-compose down
```

После запуска веб-приложение будет доступно по адресу: http://localhost:8080

### 3. Локальная разработка

#### Go сервис

```bash
# Установка зависимостей
go mod download

# Запуск
go run cmd/api/main.go
```

#### Audio Service (FastAPI)

```bash
cd audio-service
pip install -r requirements.txt
uvicorn app.main:app --reload --port 8000
```

**Важно:** Для работы STT (распознавания речи) требуется подключение к интернету, так как используется Google Speech Recognition API. Для TTS используется локальная библиотека pyttsx3.

## Зависимости

### Go
- Веб-фреймворк: Gin/Fiber/Echo
- HTTP клиент: стандартная библиотека или resty
- Конфигурация: viper или env
- Логирование: logrus или zap

### Python (FastAPI)
- fastapi
- uvicorn
- whisper (для STT) или speech_recognition
- pyttsx3 или gTTS (для TTS)
- python-multipart (для загрузки файлов)

## Интеграция с GigaChat

1. **Регистрация в GigaChat**
   - Получение Client ID и Client Secret
   - Настройка OAuth приложения

2. **Получение токена**
   - POST запрос на `/v1/oauth`
   - Использование Client ID и Secret
   - Сохранение access_token

3. **Отправка сообщений**
   - POST запрос на `/v1/chat/completions`
   - Передача access_token в заголовке
   - Отправка сообщения и контекста

4. **Обработка ответа**
   - Парсинг JSON ответа
   - Извлечение текста ответа
   - Передача в TTS сервис

## Реализованные улучшения бекенда

### ✅ Graceful Shutdown
- Корректное завершение работы сервера при получении сигналов SIGINT/SIGTERM
- Завершение активных соединений с таймаутом 10 секунд

### ✅ Валидация входных данных
- Валидация аудио файлов (размер, формат, MIME-тип)
- Валидация текстовых сообщений (длина, пустота)
- Максимальный размер файла: 10 МБ
- Максимальная длина сообщения: 5000 символов

### ✅ Retry логика для GigaChat API
- Автоматические повторы при сетевых ошибках
- До 3 попыток с экспоненциальной задержкой
- Обработка истекших токенов с автоматическим обновлением

### ✅ Структурированное логирование
- Раздельные логгеры для INFO, ERROR, WARN
- Логирование всех HTTP запросов с деталями
- Логирование операций с GigaChat и Audio Service

### ✅ Rate Limiting
- Ограничение запросов: 10 запросов в минуту с одного IP
- Автоматическая очистка старых записей
- HTTP 429 при превышении лимита

### ✅ Обработка ошибок
- Структурированные ошибки API
- Детальные сообщения об ошибках
- Recovery middleware для обработки паник

### ✅ Middleware
- CORS для работы с фронтендом
- Логирование всех запросов
- Ограничение размера запросов
- Recovery от паник

## Безопасность

- ✅ Валидация входных данных
- ✅ Rate limiting (10 запросов/минуту)
- ✅ Ограничение размера файлов
- ✅ Валидация форматов файлов
- Аутентификация пользователей (JWT) - опционально
- HTTPS в production - рекомендуется
- Безопасное хранение секретов (через .env)

## Масштабирование

- Горизонтальное масштабирование через Docker Swarm/Kubernetes
- Load balancing для API Gateway
- Отдельные инстансы для Audio Service
- Использование очередей сообщений (RabbitMQ/Kafka) для высокой нагрузки

## Тестирование

- Unit тесты для бизнес-логики
- Integration тесты для API endpoints
- Mock для GigaChat API в тестах
- Тестовые аудио файлы для STT/TTS

## Дальнейшее развитие

- Поддержка нескольких языков
- Настройка голоса и параметров TTS
- Сохранение истории диалогов в БД
- Аналитика использования
- Поддержка различных аудио форматов
- Оптимизация задержек (streaming)
- Поддержка видео звонков

