package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/Balr0g404/go-api-skeletton/internal/middleware"
)

func TestCORS_SetsHeaders(t *testing.T) {
	r := gin.New()
	r.Use(middleware.CORS())
	r.GET("/test", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "GET")
	assert.Contains(t, w.Header().Get("Access-Control-Allow-Headers"), "Authorization")
}

func TestCORS_OptionsReturns204(t *testing.T) {
	r := gin.New()
	r.Use(middleware.CORS())
	r.OPTIONS("/test", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodOptions, "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestCORS_OptionsDoesNotCallNext(t *testing.T) {
	handlerCalled := false
	r := gin.New()
	r.Use(middleware.CORS())
	r.OPTIONS("/test", func(c *gin.Context) {
		handlerCalled = true
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodOptions, "/test", nil)
	r.ServeHTTP(w, req)

	assert.False(t, handlerCalled)
}
