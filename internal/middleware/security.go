package middleware

import (
	"github.com/gin-gonic/gin"
)

func SecurityHeaders(isProd bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-XSS-Protection", "0")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		if isProd {
			c.Header("Strict-Transport-Security", "max-age=63072000; includeSubDomains; preload")
		}

		c.Next()
	}
}
