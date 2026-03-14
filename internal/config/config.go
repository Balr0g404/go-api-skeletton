package config

import (
	"fmt"
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
	Seed  SeedConfig
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

type SeedConfig struct {
	Enabled   bool
	Email     string
	Password  string
	FirstName string
	LastName  string
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
		Seed: SeedConfig{
			Enabled:   getEnvBool("SEED_ADMIN", false),
			Email:     getEnv("ADMIN_EMAIL", ""),
			Password:  getEnv("ADMIN_PASSWORD", ""),
			FirstName: getEnv("ADMIN_FIRST_NAME", "Admin"),
			LastName:  getEnv("ADMIN_LAST_NAME", "Admin"),
		},
	}
}

func (c *Config) IsProduction() bool {
	return c.App.Env == "production"
}

// knownInsecureSecrets lists placeholder values that must never reach production.
var knownInsecureSecrets = []string{
	"",
	"default-secret-change-me",
	"change-me",
	"secret",
}

// Validate checks that the configuration is safe to use in the current environment.
// It returns an error describing the first violation found.
func (c *Config) Validate() error {
	if !c.IsProduction() {
		return nil
	}

	secret := c.JWT.Secret
	for _, bad := range knownInsecureSecrets {
		if secret == bad {
			return fmt.Errorf("JWT_SECRET must be set to a strong random value in production (current value is insecure)")
		}
	}
	if len(secret) < 32 {
		return fmt.Errorf("JWT_SECRET must be at least 32 characters in production (got %d)", len(secret))
	}

	return nil
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if value, exists := os.LookupEnv(key); exists {
		return value == "true" || value == "1"
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
