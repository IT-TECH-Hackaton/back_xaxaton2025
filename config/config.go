package config

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	AppPort      string
	AppEnv       string
	DBHost       string
	DBPort       string
	DBUser       string
	DBPassword   string
	DBName       string
	DBSSLMode    string // sslmode для PostgreSQL: disable (dev), require (prod)
	JWTSecret         string
	JWTExpiration     time.Duration
	JWTRefreshExpiration time.Duration
	CookieSecure      bool
	CookieSameSite    string
	EmailHost    string
	EmailPort    int
	EmailUser    string
	EmailPassword string
	EmailFrom    string
	FrontendURL  string
	YandexClientID     string
	YandexClientSecret string
	YandexRedirectURI  string
	FakeYandexAuth     bool // Фейковая авторизация через Яндекс (для разработки)
	YandexGeocoderAPIKey string // API ключ для Яндекс.Геокодера
	CORSAllowOrigins   string // Разрешенные источники для CORS (через запятую)
	// ЮKassa (платежи)
	YooKassaShopID    string // ID магазина в ЮKassa
	YooKassaSecretKey string // Секретный ключ ЮKassa
	BackendURL        string // URL бекенда для webhook (например https://api.example.com)
	WebhookIPCheck    bool   // Проверка IP whitelist для webhook ЮKassa (в prod — true)
	SeedFull          bool   // Запуск полного сида (100+ пользователей, 100+ афиш) при старте
}

var AppConfig *Config

func LoadConfig() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	AppConfig = &Config{
		AppPort:      getEnv("APP_PORT", "8080"),
		AppEnv:       getEnv("APP_ENV", "development"),
		DBHost:       getEnv("DB_HOST", "localhost"),
		DBPort:       getEnv("DB_PORT", "5432"),
		DBUser:       getEnv("DB_USER", "postgres"),
		DBPassword:   getEnv("DB_PASSWORD", "postgres"),
		DBName:       getEnv("DB_NAME", "bekend"),
		DBSSLMode:    getDBSSLMode(),
		JWTSecret:    getEnv("JWT_SECRET", ""),
		EmailHost:    getEnv("EMAIL_HOST", "smtp.yandex.ru"),
		EmailUser:    getEnv("EMAIL_USER", ""),
		EmailPassword: getEnv("EMAIL_PASSWORD", ""),
		EmailFrom:    getEnv("EMAIL_FROM", ""),
		FrontendURL:  getEnv("FRONTEND_URL", "http://localhost:5173"),
		YandexClientID:     getEnv("YANDEX_CLIENT_ID", ""),
		YandexClientSecret: getEnv("YANDEX_CLIENT_SECRET", ""),
		YandexRedirectURI:  getEnv("YANDEX_REDIRECT_URI", "http://localhost:8081/api/auth/yandex/callback"),
		FakeYandexAuth:     getEnv("FAKE_YANDEX_AUTH", "false") == "true",
		YandexGeocoderAPIKey: getEnv("YANDEX_GEOCODER_API_KEY", ""),
		CORSAllowOrigins:   getEnv("CORS_ALLOW_ORIGINS", "http://localhost:5173"),
		YooKassaShopID:     getEnv("YOOKASSA_SHOP_ID", ""),
		YooKassaSecretKey:  getEnv("YOOKASSA_SECRET_KEY", ""),
		BackendURL:         getEnv("BACKEND_URL", "http://localhost:8081"),
		WebhookIPCheck:     getEnv("WEBHOOK_IP_CHECK", "false") == "true",
		SeedFull:           getEnv("SEED_FULL", "false") == "true",
	}

	expirationStr := getEnv("JWT_EXPIRATION", "15m")
	duration, err := time.ParseDuration(expirationStr)
	if err != nil {
		duration = 15 * time.Minute
	}
	AppConfig.JWTExpiration = duration

	refreshExpirationStr := getEnv("JWT_REFRESH_EXPIRATION", "168h")
	refreshDuration, err := time.ParseDuration(refreshExpirationStr)
	if err != nil {
		refreshDuration = 7 * 24 * time.Hour
	}
	AppConfig.JWTRefreshExpiration = refreshDuration

	AppConfig.CookieSecure = getEnv("COOKIE_SECURE", "false") == "true"
	AppConfig.CookieSameSite = getEnv("COOKIE_SAME_SITE", "lax")

	port := getEnv("EMAIL_PORT", "465")
	AppConfig.EmailPort = parseInt(port, 465)

	if AppConfig.JWTSecret == "" {
		log.Fatal("FATAL: JWT_SECRET не задан в .env — приложение не может работать без секрета JWT")
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getDBSSLMode — sslmode для PostgreSQL: в production по умолчанию require
func getDBSSLMode() string {
	if v := os.Getenv("DB_SSL_MODE"); v != "" {
		return v
	}
	if getEnv("APP_ENV", "development") == "production" {
		return "require"
	}
	return "disable"
}

func parseInt(s string, defaultValue int) int {
	var result int
	if _, err := fmt.Sscanf(s, "%d", &result); err != nil {
		return defaultValue
	}
	return result
}

