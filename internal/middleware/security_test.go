package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/Balr0g404/go-api-skeletton/internal/middleware"
)

func securityEngine(isProd bool) *gin.Engine {
	r := gin.New()
	r.Use(middleware.SecurityHeaders(isProd))
	r.GET("/test", func(c *gin.Context) { c.Status(http.StatusOK) })
	return r
}

func doSecurityRequest(r *gin.Engine) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)
	return w
}

func TestSecurityHeaders_AlwaysPresent(t *testing.T) {
	for _, isProd := range []bool{false, true} {
		w := doSecurityRequest(securityEngine(isProd))
		assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"))
		assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
		assert.Equal(t, "0", w.Header().Get("X-XSS-Protection"))
		assert.Equal(t, "strict-origin-when-cross-origin", w.Header().Get("Referrer-Policy"))
		assert.NotEmpty(t, w.Header().Get("Permissions-Policy"))
	}
}

func TestSecurityHeaders_HSTS_OnlyInProd(t *testing.T) {
	wProd := doSecurityRequest(securityEngine(true))
	assert.Contains(t, wProd.Header().Get("Strict-Transport-Security"), "max-age=63072000")
	assert.Contains(t, wProd.Header().Get("Strict-Transport-Security"), "includeSubDomains")

	wDev := doSecurityRequest(securityEngine(false))
	assert.Empty(t, wDev.Header().Get("Strict-Transport-Security"))
}

func TestSecurityHeaders_DoesNotBlockRequest(t *testing.T) {
	w := doSecurityRequest(securityEngine(false))
	assert.Equal(t, http.StatusOK, w.Code)
}
