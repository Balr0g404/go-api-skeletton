package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	App   AppConfig
	DB    DBConfig
	Redis RedisConfig
	JWT   JWTConfig
	Email EmailConfig
}

type AppConfig struct {
	Env                string
	Port               string
	BaseURL            string
	CORSAllowedOrigins []string
}

type EmailConfig struct {
	Provider     string // smtp | resend | noop
	From         string
	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	ResendAPIKey string
}

type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

type JWTConfig struct {
	Secret                 string
	ExpirationHours        int
	RefreshExpirationHours int
}

func Load() *Config {
	return &Config{
		App: AppConfig{
			Env:                getEnv("APP_ENV", "development"),
			Port:               getEnv("APP_PORT", "8080"),
			BaseURL:            getEnv("APP_BASE_URL", "http://localhost:8080"),
			CORSAllowedOrigins: strings.Split(getEnv("CORS_ALLOWED_ORIGINS", "*"), ","),
		},
		DB: DBConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "appuser"),
			Password: getEnv("DB_PASSWORD", "apppassword"),
			Name:     getEnv("DB_NAME", "appdb"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvInt("REDIS_DB", 0),
		},
		JWT: JWTConfig{
			Secret:                 getEnv("JWT_SECRET", "default-secret-change-me"),
			ExpirationHours:        getEnvInt("JWT_EXPIRATION_HOURS", 24),
			RefreshExpirationHours: getEnvInt("JWT_REFRESH_EXPIRATION_HOURS", 168),
		},
		Email: EmailConfig{
			Provider:     getEnv("EMAIL_PROVIDER", "noop"),
			From:         getEnv("EMAIL_FROM", "noreply@example.com"),
			SMTPHost:     getEnv("SMTP_HOST", "localhost"),
			SMTPPort:     getEnvInt("SMTP_PORT", 587),
			SMTPUsername: getEnv("SMTP_USERNAME", ""),
			SMTPPassword: getEnv("SMTP_PASSWORD", ""),
			ResendAPIKey: getEnv("RESEND_API_KEY", ""),
		},
	}
}

func (c *Config) IsProduction() bool {
	return c.App.Env == "production"
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return fallback
}
