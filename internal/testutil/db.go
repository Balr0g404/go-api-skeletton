//go:build integration

package testutil

import (
	"net"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/Balr0g404/go-api-skeletton/internal/config"
	"github.com/Balr0g404/go-api-skeletton/internal/database"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func NewTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping integration test")
	}

	cfg := parseDSN(t, dsn)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	database.RunMigrations(cfg)

	t.Cleanup(func() {
		db.Exec("TRUNCATE TABLE users RESTART IDENTITY CASCADE")
	})

	return db
}

func parseDSN(t *testing.T, dsn string) *config.DBConfig {
	t.Helper()
	u, err := url.Parse(dsn)
	require.NoError(t, err)

	password, _ := u.User.Password()
	host, port, _ := net.SplitHostPort(u.Host)
	sslmode := u.Query().Get("sslmode")
	if sslmode == "" {
		sslmode = "disable"
	}

	return &config.DBConfig{
		Host:     host,
		Port:     port,
		User:     u.User.Username(),
		Password: password,
		Name:     strings.TrimPrefix(u.Path, "/"),
		SSLMode:  sslmode,
	}
}
