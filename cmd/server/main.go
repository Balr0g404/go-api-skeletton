package main

import (
	_ "github.com/Balr0g404/go-api-skeletton/docs"
	"github.com/Balr0g404/go-api-skeletton/internal/config"
	"github.com/Balr0g404/go-api-skeletton/internal/database"
	"github.com/Balr0g404/go-api-skeletton/internal/repositories"
	"github.com/Balr0g404/go-api-skeletton/internal/router"
	"github.com/Balr0g404/go-api-skeletton/internal/services"
	"github.com/Balr0g404/go-api-skeletton/pkg/auth"
	"github.com/Balr0g404/go-api-skeletton/pkg/logger"
	"github.com/rs/zerolog/log"
)

// @title           Go API Template
// @version         1.0
// @description     REST API template with authentication and authorization

// @host      localhost:8080
// @BasePath  /api/v1

// @securityDefinitions.apikey  BearerAuth
// @in                          header
// @name                        Authorization
// @description                 Enter "Bearer {token}"

func main() {
	cfg := config.Load()
	logger.Init(cfg.IsProduction())

	db := database.NewPostgres(&cfg.DB, cfg.IsProduction())

	database.RunMigrations(&cfg.DB)
	database.Seed(db)

	redisClient := database.NewRedis(&cfg.Redis)

	jwtManager := auth.NewJWTManager(
		cfg.JWT.Secret,
		cfg.JWT.ExpirationHours,
		cfg.JWT.RefreshExpirationHours,
	)

	userRepo := repositories.NewUserRepository(db)
	authService := services.NewAuthService(userRepo, jwtManager, redisClient)

	r := router.Setup(jwtManager, authService, redisClient, cfg.IsProduction())

	log.Info().Str("port", cfg.App.Port).Str("env", cfg.App.Env).Msg("starting server")
	if err := r.Run(":" + cfg.App.Port); err != nil {
		log.Fatal().Err(err).Msg("failed to start server")
	}
}