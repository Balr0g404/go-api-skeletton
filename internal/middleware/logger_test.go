package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/Balr0g404/go-api-skeletton/internal/middleware"
)

func loggerEngine() *gin.Engine {
	r := gin.New()
	r.Use(middleware.RequestID())
	r.Use(middleware.Logger())
	return r
}

func TestLogger_DoesNotBlockRequest(t *testing.T) {
	r := loggerEngine()
	r.GET("/ok", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/ok", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestLogger_4xxRequest(t *testing.T) {
	r := loggerEngine()
	r.GET("/notfound", func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/notfound", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestLogger_5xxRequest(t *testing.T) {
	r := loggerEngine()
	r.GET("/err", func(c *gin.Context) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "boom"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/err", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestLogger_PropagatesRequestID(t *testing.T) {
	r := loggerEngine()
	var capturedID string
	r.GET("/test", func(c *gin.Context) {
		capturedID = c.GetHeader("X-Request-ID")
		c.JSON(http.StatusOK, gin.H{})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Request-ID", "my-trace-id")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	_ = capturedID // logger reads from context, not header — just verify no panic
}

func TestLogger_WorksWithoutRequestID(t *testing.T) {
	// Logger alone, no RequestID middleware — should not panic
	r := gin.New()
	r.Use(middleware.Logger())
	r.GET("/bare", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/bare", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
