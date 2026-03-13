package database

import (
	"errors"
	"fmt"

	"github.com/Balr0g404/go-api-skeletton/internal/config"
	"github.com/Balr0g404/go-api-skeletton/migrations"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/rs/zerolog/log"
)

func RunMigrations(cfg *config.DBConfig) {
	source, err := iofs.New(migrations.FS, ".")
	if err != nil {
		log.Fatal().Err(err).Msg("migrations: failed to load source")
	}

	dsn := fmt.Sprintf(
		"pgx5://%s:%s@%s:%s/%s?sslmode=%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name, cfg.SSLMode,
	)

	m, err := migrate.NewWithSourceInstance("iofs", source, dsn)
	if err != nil {
		log.Fatal().Err(err).Msg("migrations: failed to create migrator")
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		log.Fatal().Err(err).Msg("migrations: failed to apply")
	}

	log.Info().Msg("migrations: up to date")
}
