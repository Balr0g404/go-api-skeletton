package router

import (
	"time"

	"github.com/Balr0g404/go-api-skeletton/internal/handlers"
	"github.com/Balr0g404/go-api-skeletton/internal/middleware"
	"github.com/Balr0g404/go-api-skeletton/internal/models"
	"github.com/Balr0g404/go-api-skeletton/internal/services"
	"github.com/Balr0g404/go-api-skeletton/pkg/auth"
	"github.com/Balr0g404/go-api-skeletton/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func Setup(
	jwtManager *auth.JWTManager,
	authService *services.AuthService,
	redisClient *redis.Client,
	isProd bool,
	allowedOrigins []string,
) *gin.Engine {
	if isProd {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(middleware.RequestID())
	r.Use(middleware.Recovery())
	r.Use(middleware.Logger())
	r.Use(middleware.Timeout(30 * time.Second))
	r.Use(middleware.SecurityHeaders(isProd))
	r.Use(middleware.CORS(allowedOrigins))
	r.Use(middleware.RateLimit(redisClient, 100, time.Minute))

	authHandler := handlers.NewAuthHandler(authService)

	// @Summary      Health check
	// @Tags         System
	// @Produce      json
	// @Success      200  {object}  response.APIResponse{data=handlers.HealthResponse}
	// @Router       /health [get]
	r.GET("/health", func(c *gin.Context) {
		response.OK(c, gin.H{"status": "ok"})
	})

	if !isProd {
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	api := r.Group("/api/v1")
	{
		public := api.Group("/auth")
		{
			public.POST("/register", authHandler.Register)
			public.POST("/login", authHandler.Login)
			public.POST("/refresh", authHandler.RefreshToken)
			public.POST("/forgot-password", authHandler.ForgotPassword)
			public.POST("/reset-password", authHandler.ResetPassword)
		}

		protected := api.Group("")
		protected.Use(middleware.AuthRequired(jwtManager, authService))
		{
			protected.POST("/auth/logout", authHandler.Logout)
			protected.GET("/profile", authHandler.GetProfile)
			protected.PUT("/profile", authHandler.UpdateProfile)
			protected.PUT("/profile/password", authHandler.ChangePassword)
		}

		admin := api.Group("/admin")
		admin.Use(middleware.AuthRequired(jwtManager, authService))
		admin.Use(middleware.RoleRequired(models.RoleAdmin))
		{
			admin.GET("/users", authHandler.ListUsers)
			admin.GET("/users/cursor", authHandler.ListUsersCursor)
			admin.PUT("/users/:id/role", authHandler.SetUserRole)
		}
	}

	return r
}
