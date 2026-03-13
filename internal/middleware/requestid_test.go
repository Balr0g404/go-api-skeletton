package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/Balr0g404/go-api-skeletton/internal/middleware"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func newEngine(handlers ...gin.HandlerFunc) *gin.Engine {
	r := gin.New()
	r.Use(handlers...)
	r.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	return r
}

func TestRequestID_GeneratesIDWhenAbsent(t *testing.T) {
	r := newEngine(middleware.RequestID())

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	id := w.Header().Get("X-Request-ID")
	assert.NotEmpty(t, id)
}

func TestRequestID_UsesExistingHeader(t *testing.T) {
	r := newEngine(middleware.RequestID())

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Request-ID", "my-custom-id")
	r.ServeHTTP(w, req)

	assert.Equal(t, "my-custom-id", w.Header().Get("X-Request-ID"))
}

func TestRequestID_SetsContextKey(t *testing.T) {
	r := gin.New()
	r.Use(middleware.RequestID())
	r.GET("/test", func(c *gin.Context) {
		id, exists := c.Get(middleware.RequestIDKey)
		assert.True(t, exists)
		assert.NotEmpty(t, id)
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)
}

func TestRequestID_UniquePerRequest(t *testing.T) {
	r := newEngine(middleware.RequestID())

	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w1, req1)

	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w2, req2)

	assert.NotEqual(t, w1.Header().Get("X-Request-ID"), w2.Header().Get("X-Request-ID"))
}
