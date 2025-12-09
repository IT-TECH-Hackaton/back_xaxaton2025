package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Server       ServerConfig
	GigaChat     GigaChatConfig
	AudioService AudioServiceConfig
	Database     DatabaseConfig
	JWT          JWTConfig
	Email        EmailConfig
}

type ServerConfig struct {
	Port       string
	Host       string
	SessionTTL time.Duration
}

type GigaChatConfig struct {
	ClientID         string
	ClientSecret     string
	AuthorizationKey string
	Scope            string
	AuthURL          string
	APIURL           string
}

type AudioServiceConfig struct {
	URL string
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

type JWTConfig struct {
	Secret          string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
}

type EmailConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	From     string
}

func Load() (*Config, error) {
	sessionTTL := 30 * time.Minute
	if ttlStr := getEnv("SESSION_TTL", ""); ttlStr != "" {
		if parsed, err := time.ParseDuration(ttlStr); err == nil {
			sessionTTL = parsed
		}
	}

	return &Config{
		Server: ServerConfig{
			Port:       getEnv("API_PORT", "8080"),
			Host:       getEnv("API_HOST", "0.0.0.0"),
			SessionTTL: sessionTTL,
		},
		GigaChat: GigaChatConfig{
			ClientID:         getEnv("GIGACHAT_CLIENT_ID", "019a81d2-9f7c-7429-a7eb-f240038d4d22"),
			ClientSecret:     getEnv("GIGACHAT_CLIENT_SECRET", "9fc30b5d-f451-4963-8495-7da27ef39ef1"),
			AuthorizationKey: getEnv("GIGACHAT_AUTHORIZATION_KEY", "MDE5YTgxZDItOWY3Yy03NDI5LWE3ZWItZjI0MDAzOGQ0ZDIyOjlmYzMwYjVkLWY0NTEtNDk2My04NDk1LTdkYTI3ZWYzOWVmMQ=="),
			Scope:            getEnv("GIGACHAT_SCOPE", "GIGACHAT_API_PERS"),
			AuthURL:          getEnv("GIGACHAT_AUTH_URL", "https://ngw.devices.sberbank.ru:9443/api/v2/oauth"),
			APIURL:           getEnv("GIGACHAT_API_URL", "https://gigachat.devices.sberbank.ru/api/v1"),
		},
		AudioService: AudioServiceConfig{
			URL: getEnv("AUDIO_SERVICE_URL", "http://localhost:8000"),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "postgres"),
			DBName:   getEnv("DB_NAME", "golos"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		JWT: JWTConfig{
			Secret:          getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
			AccessTokenTTL:  parseDuration(getEnv("JWT_ACCESS_TTL", "15m"), 15*time.Minute),
			RefreshTokenTTL: parseDuration(getEnv("JWT_REFRESH_TTL", "168h"), 168*time.Hour),
		},
		Email: EmailConfig{
			Host:     getEnv("EMAIL_HOST", "smtp.gmail.com"),
			Port:     parseInt(getEnv("EMAIL_PORT", "587"), 587),
			User:     getEnv("EMAIL_USER", ""),
			Password: getEnv("EMAIL_PASSWORD", ""),
			From:     getEnv("EMAIL_FROM", ""),
		},
	}, nil
}

func parseDuration(s string, defaultValue time.Duration) time.Duration {
	if s == "" {
		return defaultValue
	}
	if d, err := time.ParseDuration(s); err == nil {
		return d
	}
	return defaultValue
}

func parseInt(s string, defaultValue int) int {
	if s == "" {
		return defaultValue
	}
	if i, err := strconv.Atoi(s); err == nil {
		return i
	}
	return defaultValue
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
