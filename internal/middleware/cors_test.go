package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/Balr0g404/go-api-skeletton/internal/middleware"
)

func corsEngine(origins []string) *gin.Engine {
	r := gin.New()
	r.Use(middleware.CORS(origins))
	r.GET("/test", func(c *gin.Context) { c.Status(http.StatusOK) })
	r.OPTIONS("/test", func(c *gin.Context) { c.Status(http.StatusOK) })
	return r
}

func TestCORS_Wildcard_AllowsAll(t *testing.T) {
	r := corsEngine([]string{"*"})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "https://any-site.com")
	r.ServeHTTP(w, req)

	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "GET")
	assert.Contains(t, w.Header().Get("Access-Control-Allow-Headers"), "Authorization")
}

func TestCORS_AllowedOrigin_Matches(t *testing.T) {
	r := corsEngine([]string{"https://app.example.com", "https://admin.example.com"})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "https://app.example.com")
	r.ServeHTTP(w, req)

	assert.Equal(t, "https://app.example.com", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "Origin", w.Header().Get("Vary"))
}

func TestCORS_AllowedOrigin_NoMatch(t *testing.T) {
	r := corsEngine([]string{"https://app.example.com"})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "https://evil.com")
	r.ServeHTTP(w, req)

	assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORS_NoOriginHeader(t *testing.T) {
	r := corsEngine([]string{"https://app.example.com"})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORS_OptionsReturns204(t *testing.T) {
	r := corsEngine([]string{"*"})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodOptions, "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestCORS_OptionsDoesNotCallNext(t *testing.T) {
	handlerCalled := false
	r := gin.New()
	r.Use(middleware.CORS([]string{"*"}))
	r.OPTIONS("/test", func(c *gin.Context) {
		handlerCalled = true
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodOptions, "/test", nil)
	r.ServeHTTP(w, req)

	assert.False(t, handlerCalled)
}
