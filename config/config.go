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
	JWTSecret    string
	JWTExpiration time.Duration
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
		JWTSecret:    getEnv("JWT_SECRET", "change-me-in-production"),
		EmailHost:    getEnv("EMAIL_HOST", "smtp.gmail.com"),
		EmailUser:    getEnv("EMAIL_USER", ""),
		EmailPassword: getEnv("EMAIL_PASSWORD", ""),
		EmailFrom:    getEnv("EMAIL_FROM", ""),
		FrontendURL:  getEnv("FRONTEND_URL", "http://localhost:5173"),
		YandexClientID:     getEnv("YANDEX_CLIENT_ID", ""),
		YandexClientSecret: getEnv("YANDEX_CLIENT_SECRET", ""),
		YandexRedirectURI:  getEnv("YANDEX_REDIRECT_URI", "http://localhost:8081/api/auth/yandex/callback"),
		FakeYandexAuth:     getEnv("FAKE_YANDEX_AUTH", "false") == "true",
	}

	expirationStr := getEnv("JWT_EXPIRATION", "24h")
	duration, err := time.ParseDuration(expirationStr)
	if err != nil {
		duration = 24 * time.Hour
	}
	AppConfig.JWTExpiration = duration

	port := getEnv("EMAIL_PORT", "587")
	AppConfig.EmailPort = parseInt(port, 587)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func parseInt(s string, defaultValue int) int {
	var result int
	if _, err := fmt.Sscanf(s, "%d", &result); err != nil {
		return defaultValue
	}
	return result
}

