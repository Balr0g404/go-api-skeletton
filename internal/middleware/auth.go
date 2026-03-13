package middleware

import (
	"strings"

	"github.com/Balr0g404/go-api-skeletton/internal/models"
	"github.com/Balr0g404/go-api-skeletton/internal/services"
	"github.com/Balr0g404/go-api-skeletton/pkg/auth"
	"github.com/Balr0g404/go-api-skeletton/pkg/response"
	"github.com/gin-gonic/gin"
)

func AuthRequired(jwtManager *auth.JWTManager, authService *services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			response.Unauthorized(c, "authorization header required")
			c.Abort()
			return
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			response.Unauthorized(c, "invalid authorization format")
			c.Abort()
			return
		}

		token := parts[1]

		if authService.IsTokenBlacklisted(token) {
			response.Unauthorized(c, "token has been revoked")
			c.Abort()
			return
		}

		claims, err := jwtManager.ValidateToken(token, auth.AccessToken)
		if err != nil {
			response.Unauthorized(c, "invalid or expired token")
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Set("user_role", claims.Role)
		c.Set("access_token", token)

		c.Next()
	}
}

func RoleRequired(roles ...models.Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get("user_role")
		if !exists {
			response.Unauthorized(c, "authentication required")
			c.Abort()
			return
		}

		role := models.Role(userRole.(string))
		for _, allowed := range roles {
			if role == allowed {
				c.Next()
				return
			}
		}

		response.Forbidden(c, "insufficient permissions")
		c.Abort()
	}
}

func GetUserID(c *gin.Context) uint {
	id, _ := c.Get("user_id")
	return id.(uint)
}

func GetUserRole(c *gin.Context) string {
	role, _ := c.Get("user_role")
	return role.(string)
}

func GetAccessToken(c *gin.Context) string {
	token, _ := c.Get("access_token")
	return token.(string)
}
