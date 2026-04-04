package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/Balr0g404/go-api-skeletton/docs"
	"github.com/Balr0g404/go-api-skeletton/internal/config"
	"github.com/Balr0g404/go-api-skeletton/internal/database"
	"github.com/Balr0g404/go-api-skeletton/internal/repositories"
	"github.com/Balr0g404/go-api-skeletton/internal/router"
	"github.com/Balr0g404/go-api-skeletton/internal/services"
	"github.com/Balr0g404/go-api-skeletton/pkg/auth"
	"github.com/Balr0g404/go-api-skeletton/pkg/email"
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

	if err := cfg.Validate(); err != nil {
		log.Fatal().Err(err).Msg("invalid configuration")
	}

	db, err := database.NewPostgres(&cfg.DB, cfg.IsProduction())
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}

	database.RunMigrations(&cfg.DB)
	database.Seed(db, cfg.IsProduction(), cfg.Seed)

	redisClient, err := database.NewRedis(&cfg.Redis)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to redis")
	}

	jwtManager := auth.NewJWTManager(
		cfg.JWT.Secret,
		cfg.JWT.ExpirationHours,
		cfg.JWT.RefreshExpirationHours,
	)

	mailer, err := email.New(email.Config{
		Provider:     cfg.Email.Provider,
		From:         cfg.Email.From,
		SMTPHost:     cfg.Email.SMTPHost,
		SMTPPort:     cfg.Email.SMTPPort,
		SMTPUsername: cfg.Email.SMTPUsername,
		SMTPPassword: cfg.Email.SMTPPassword,
		ResendAPIKey: cfg.Email.ResendAPIKey,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize email provider")
	}

	userRepo := repositories.NewUserRepository(db)
	authService := services.NewAuthService(userRepo, jwtManager, redisClient, mailer, cfg.App.BaseURL)

	r := router.Setup(jwtManager, authService, redisClient, cfg.IsProduction(), cfg.App.CORSAllowedOrigins)

	srv := &http.Server{
		Addr:    ":" + cfg.App.Port,
		Handler: r,
	}

	go func() {
		log.Info().Str("port", cfg.App.Port).Str("env", cfg.App.Env).Msg("starting server")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("failed to start server")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("shutting down server")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal().Err(err).Msg("forced shutdown")
	}
	log.Info().Msg("server stopped")
}
