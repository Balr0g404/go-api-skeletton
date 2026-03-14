package middleware_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Balr0g404/go-api-skeletton/internal/middleware"
)

func recoveryEngine() *gin.Engine {
	r := gin.New()
	r.Use(middleware.RequestID())
	r.Use(middleware.Recovery())
	return r
}

func TestRecovery_CatchesPanic(t *testing.T) {
	r := recoveryEngine()
	r.GET("/panic", func(c *gin.Context) {
		panic("something went wrong")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/panic", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestRecovery_Returns500JSON(t *testing.T) {
	r := recoveryEngine()
	r.GET("/panic", func(c *gin.Context) {
		panic("boom")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/panic", nil)
	r.ServeHTTP(w, req)

	var body map[string]interface{}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Equal(t, false, body["success"])
	assert.Equal(t, "internal server error", body["error"])
}

func TestRecovery_CatchesNilPanic(t *testing.T) {
	r := recoveryEngine()
	r.GET("/panic", func(c *gin.Context) {
		panic(nil)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/panic", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestRecovery_DoesNotAffectNormalRequests(t *testing.T) {
	r := recoveryEngine()
	r.GET("/ok", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/ok", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRecovery_CatchesErrorPanic(t *testing.T) {
	r := recoveryEngine()
	r.GET("/panic", func(c *gin.Context) {
		var s []int
		_ = s[0]
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/panic", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestRecovery_SetsRequestIDInLog(t *testing.T) {
	r := recoveryEngine()
	r.GET("/panic", func(c *gin.Context) {
		panic("test")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/panic", nil)
	req.Header.Set("X-Request-ID", "test-request-id-123")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "test-request-id-123", w.Header().Get("X-Request-ID"))
}
