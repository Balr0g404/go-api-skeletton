package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Balr0g404/go-api-skeletton/internal/mocks"
	"github.com/Balr0g404/go-api-skeletton/internal/middleware"
	"github.com/Balr0g404/go-api-skeletton/internal/models"
	"github.com/Balr0g404/go-api-skeletton/internal/services"
	"github.com/Balr0g404/go-api-skeletton/pkg/auth"
)

type authTestSetup struct {
	engine     *gin.Engine
	jwtManager *auth.JWTManager
	svc        *services.AuthService
	mr         *miniredis.Miniredis
}

func newAuthTestSetup(t *testing.T) authTestSetup {
	t.Helper()

	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(func() { mr.Close() })

	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	jwtMgr := auth.NewJWTManager("test-secret", 1, 24)
	repo := &mocks.UserRepository{}
	svc := services.NewAuthService(repo, jwtMgr, client)

	r := gin.New()
	r.Use(middleware.AuthRequired(jwtMgr, svc))
	r.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"user_id": middleware.GetUserID(c),
			"role":    middleware.GetUserRole(c),
		})
	})

	return authTestSetup{engine: r, jwtManager: jwtMgr, svc: svc, mr: mr}
}

func TestAuthRequired_NoHeader(t *testing.T) {
	s := newAuthTestSetup(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	s.engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthRequired_InvalidFormat(t *testing.T) {
	s := newAuthTestSetup(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Token abc123")
	s.engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthRequired_MissingBearerPrefix(t *testing.T) {
	s := newAuthTestSetup(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "justtoken")
	s.engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthRequired_InvalidToken(t *testing.T) {
	s := newAuthTestSetup(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.here")
	s.engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthRequired_BlacklistedToken(t *testing.T) {
	s := newAuthTestSetup(t)

	pair, err := s.jwtManager.GenerateTokenPair(1, "user@example.com", "user")
	require.NoError(t, err)

	s.svc.Logout(pair.AccessToken, "")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	s.engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthRequired_ValidToken(t *testing.T) {
	s := newAuthTestSetup(t)

	pair, err := s.jwtManager.GenerateTokenPair(42, "user@example.com", "user")
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	s.engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthRequired_RefreshTokenRejected(t *testing.T) {
	s := newAuthTestSetup(t)

	pair, err := s.jwtManager.GenerateTokenPair(1, "user@example.com", "user")
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+pair.RefreshToken)
	s.engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRoleRequired_AllowedRole(t *testing.T) {
	r := gin.New()
	r.GET("/admin", func(c *gin.Context) {
		c.Set("user_role", string(models.RoleAdmin))
	}, middleware.RoleRequired(models.RoleAdmin), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/admin", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRoleRequired_ForbiddenRole(t *testing.T) {
	r := gin.New()
	r.GET("/admin", func(c *gin.Context) {
		c.Set("user_role", string(models.RoleUser))
	}, middleware.RoleRequired(models.RoleAdmin), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/admin", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestRoleRequired_NoRoleInContext(t *testing.T) {
	r := gin.New()
	r.GET("/admin", middleware.RoleRequired(models.RoleAdmin), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/admin", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRoleRequired_MultipleAllowedRoles(t *testing.T) {
	r := gin.New()
	r.GET("/resource", func(c *gin.Context) {
		c.Set("user_role", string(models.RoleUser))
	}, middleware.RoleRequired(models.RoleUser, models.RoleAdmin), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/resource", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
