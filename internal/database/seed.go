package database

import (
	"github.com/Balr0g404/go-api-skeletton/internal/config"
	"github.com/Balr0g404/go-api-skeletton/internal/models"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

// Seed creates the initial admin user when SEED_ADMIN=true.
// It is intentionally disabled in production to avoid running
// with known or default credentials.
func Seed(db *gorm.DB, isProd bool, cfg config.SeedConfig) {
	if !cfg.Enabled {
		return
	}

	if isProd {
		log.Warn().Msg("seed skipped: SEED_ADMIN=true is not allowed in production")
		return
	}

	if cfg.Email == "" || cfg.Password == "" {
		log.Fatal().Msg("seed requires ADMIN_EMAIL and ADMIN_PASSWORD to be set")
	}

	var count int64
	db.Model(&models.User{}).Where("email = ?", cfg.Email).Count(&count)
	if count > 0 {
		log.Info().Str("email", cfg.Email).Msg("seed skipped: admin user already exists")
		return
	}

	admin := &models.User{
		Email:     cfg.Email,
		FirstName: cfg.FirstName,
		LastName:  cfg.LastName,
		Role:      models.RoleAdmin,
		Active:    true,
	}

	if err := admin.SetPassword(cfg.Password); err != nil {
		log.Fatal().Err(err).Msg("failed to hash admin password")
	}

	if err := db.Create(admin).Error; err != nil {
		log.Fatal().Err(err).Msg("failed to seed admin user")
	}

	log.Info().Str("email", cfg.Email).Msg("admin user seeded")
}
