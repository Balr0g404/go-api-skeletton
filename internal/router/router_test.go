package router_test

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
	"github.com/Balr0g404/go-api-skeletton/internal/router"
	"github.com/Balr0g404/go-api-skeletton/internal/services"
	"github.com/Balr0g404/go-api-skeletton/pkg/auth"
	"github.com/stretchr/testify/mock"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func newTestRouter(t *testing.T, isProd bool) *gin.Engine {
	t.Helper()

	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(func() { mr.Close() })

	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	jwtMgr := auth.NewJWTManager("test-secret", 1, 24)
	repo := &mocks.UserRepository{}
	mailer := &mocks.EmailSender{}
	mailer.On("Send", mock.Anything).Return(nil)
	svc := services.NewAuthService(repo, jwtMgr, client, mailer, "http://localhost:8080")

	return router.Setup(jwtMgr, svc, client, isProd, []string{"*"})
}

// ─── Health check ─────────────────────────────────────────────────────────────

func TestRouter_HealthCheck_Returns200(t *testing.T) {
	r := newTestRouter(t, false)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/health", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// ─── Auth routes reachable ────────────────────────────────────────────────────

func TestRouter_Register_RouteExists(t *testing.T) {
	r := newTestRouter(t, false)

	// POST with no body → 400 (validation), not 404
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/register", nil)
	r.ServeHTTP(w, req)

	assert.NotEqual(t, http.StatusNotFound, w.Code)
}

func TestRouter_Login_RouteExists(t *testing.T) {
	r := newTestRouter(t, false)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	r.ServeHTTP(w, req)

	assert.NotEqual(t, http.StatusNotFound, w.Code)
}

func TestRouter_Refresh_RouteExists(t *testing.T) {
	r := newTestRouter(t, false)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	r.ServeHTTP(w, req)

	assert.NotEqual(t, http.StatusNotFound, w.Code)
}

// ─── Protected routes require auth ────────────────────────────────────────────

func TestRouter_Profile_RequiresAuth(t *testing.T) {
	r := newTestRouter(t, false)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/profile", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRouter_Logout_RequiresAuth(t *testing.T) {
	r := newTestRouter(t, false)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ─── Admin routes require auth ────────────────────────────────────────────────

func TestRouter_AdminUsers_RequiresAuth(t *testing.T) {
	r := newTestRouter(t, false)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/admin/users", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRouter_AdminUsersCursor_RequiresAuth(t *testing.T) {
	r := newTestRouter(t, false)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/admin/users/cursor", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ─── Swagger only in dev ──────────────────────────────────────────────────────
// Note: the swagger route (/swagger/*any) is only registered in dev mode.
// In prod, the route is absent so gin returns 404.

func TestRouter_Swagger_NotRegisteredInProd(t *testing.T) {
	r := newTestRouter(t, true)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/swagger/index.html", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ─── Security headers present ─────────────────────────────────────────────────

func TestRouter_SecurityHeaders_OnHealthCheck(t *testing.T) {
	r := newTestRouter(t, false)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/health", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"))
	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
}

// ─── RequestID header present ────────────────────────────────────────────────

func TestRouter_RequestID_InResponse(t *testing.T) {
	r := newTestRouter(t, false)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/health", nil)
	r.ServeHTTP(w, req)

	assert.NotEmpty(t, w.Header().Get("X-Request-ID"))
}
