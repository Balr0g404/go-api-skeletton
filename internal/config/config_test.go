package config_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Balr0g404/go-api-skeletton/internal/config"
)

func TestLoad_Defaults(t *testing.T) {
	cfg := config.Load()

	assert.Equal(t, "development", cfg.App.Env)
	assert.Equal(t, "8080", cfg.App.Port)
	assert.Equal(t, []string{"*"}, cfg.App.CORSAllowedOrigins)
	assert.Equal(t, "localhost", cfg.DB.Host)
	assert.Equal(t, "5432", cfg.DB.Port)
	assert.Equal(t, "appuser", cfg.DB.User)
	assert.Equal(t, "appdb", cfg.DB.Name)
	assert.Equal(t, "disable", cfg.DB.SSLMode)
	assert.Equal(t, "localhost", cfg.Redis.Host)
	assert.Equal(t, "6379", cfg.Redis.Port)
	assert.Equal(t, 0, cfg.Redis.DB)
	assert.Equal(t, 24, cfg.JWT.ExpirationHours)
	assert.Equal(t, 168, cfg.JWT.RefreshExpirationHours)
}

func TestLoad_FromEnv(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("APP_PORT", "9090")
	t.Setenv("DB_HOST", "db.internal")
	t.Setenv("DB_PORT", "5433")
	t.Setenv("DB_USER", "myuser")
	t.Setenv("DB_PASSWORD", "mypassword")
	t.Setenv("DB_NAME", "mydb")
	t.Setenv("DB_SSLMODE", "require")
	t.Setenv("REDIS_HOST", "redis.internal")
	t.Setenv("REDIS_PORT", "6380")
	t.Setenv("REDIS_DB", "1")
	t.Setenv("JWT_SECRET", "supersecret")
	t.Setenv("JWT_EXPIRATION_HOURS", "2")
	t.Setenv("JWT_REFRESH_EXPIRATION_HOURS", "48")

	cfg := config.Load()

	assert.Equal(t, "production", cfg.App.Env)
	assert.Equal(t, "9090", cfg.App.Port)
	assert.Equal(t, "db.internal", cfg.DB.Host)
	assert.Equal(t, "5433", cfg.DB.Port)
	assert.Equal(t, "myuser", cfg.DB.User)
	assert.Equal(t, "mypassword", cfg.DB.Password)
	assert.Equal(t, "mydb", cfg.DB.Name)
	assert.Equal(t, "require", cfg.DB.SSLMode)
	assert.Equal(t, "redis.internal", cfg.Redis.Host)
	assert.Equal(t, "6380", cfg.Redis.Port)
	assert.Equal(t, 1, cfg.Redis.DB)
	assert.Equal(t, "supersecret", cfg.JWT.Secret)
	assert.Equal(t, 2, cfg.JWT.ExpirationHours)
	assert.Equal(t, 48, cfg.JWT.RefreshExpirationHours)
}

func TestLoad_InvalidIntFallsBackToDefault(t *testing.T) {
	t.Setenv("REDIS_DB", "not-an-int")
	t.Setenv("JWT_EXPIRATION_HOURS", "abc")

	cfg := config.Load()

	assert.Equal(t, 0, cfg.Redis.DB)
	assert.Equal(t, 24, cfg.JWT.ExpirationHours)
}

func TestIsProduction_True(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	cfg := config.Load()
	assert.True(t, cfg.IsProduction())
}

func TestIsProduction_False(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	cfg := config.Load()
	assert.False(t, cfg.IsProduction())
}

func TestIsProduction_Default(t *testing.T) {
	cfg := config.Load()
	assert.False(t, cfg.IsProduction())
}

func TestLoad_CORSAllowedOrigins_Single(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://app.example.com")
	cfg := config.Load()
	assert.Equal(t, []string{"https://app.example.com"}, cfg.App.CORSAllowedOrigins)
}

func TestLoad_CORSAllowedOrigins_Multiple(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://app.example.com,https://admin.example.com")
	cfg := config.Load()
	assert.Equal(t, []string{"https://app.example.com", "https://admin.example.com"}, cfg.App.CORSAllowedOrigins)
}

// ── Validate ─────────────────────────────────────────────────────────────────

func TestValidate_DevAlwaysPasses(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("JWT_SECRET", "") // even empty is fine in dev
	cfg := config.Load()
	assert.NoError(t, cfg.Validate())
}

func TestValidate_ProdWithStrongSecret(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("JWT_SECRET", "a-very-strong-secret-that-is-long-enough-for-prod")
	cfg := config.Load()
	assert.NoError(t, cfg.Validate())
}

func TestValidate_ProdWithEmptySecret(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("JWT_SECRET", "")
	cfg := config.Load()
	assert.Error(t, cfg.Validate())
}

func TestValidate_ProdWithDefaultSecret(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("JWT_SECRET", "default-secret-change-me")
	cfg := config.Load()
	assert.Error(t, cfg.Validate())
}

func TestValidate_ProdWithShortSecret(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("JWT_SECRET", "tooshort")
	cfg := config.Load()
	assert.Error(t, cfg.Validate())
}
