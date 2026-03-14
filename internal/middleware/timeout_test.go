package middleware_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Balr0g404/go-api-skeletton/internal/middleware"
)

func timeoutEngine(d time.Duration, handler gin.HandlerFunc) *gin.Engine {
	r := gin.New()
	r.Use(middleware.Timeout(d))
	r.GET("/test", handler)
	return r
}

func TestTimeout_FastHandlerCompletes(t *testing.T) {
	r := timeoutEngine(100*time.Millisecond, func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTimeout_SlowHandlerReturns503(t *testing.T) {
	r := timeoutEngine(20*time.Millisecond, func(c *gin.Context) {
		time.Sleep(200 * time.Millisecond)
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	var body map[string]interface{}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Equal(t, false, body["success"])
	assert.Equal(t, "request timeout", body["error"])
}

func TestTimeout_ContextDeadlinePropagated(t *testing.T) {
	var deadline time.Time
	var ok bool

	r := timeoutEngine(500*time.Millisecond, func(c *gin.Context) {
		deadline, ok = c.Request.Context().Deadline()
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	assert.True(t, ok, "request context should have a deadline")
	assert.WithinDuration(t, time.Now().Add(500*time.Millisecond), deadline, 600*time.Millisecond)
}

func TestTimeout_ContextCancelledOnTimeout(t *testing.T) {
	ctxDone := make(chan struct{})

	r := timeoutEngine(20*time.Millisecond, func(c *gin.Context) {
		select {
		case <-c.Request.Context().Done():
			close(ctxDone)
		case <-time.After(500 * time.Millisecond):
		}
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	select {
	case <-ctxDone:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected context to be cancelled, but it was not")
	}
}

func TestTimeout_RespectsIncomingContextCancellation(t *testing.T) {
	r := timeoutEngine(500*time.Millisecond, func(c *gin.Context) {
		select {
		case <-c.Request.Context().Done():
			assert.ErrorIs(t, c.Request.Context().Err(), context.Canceled)
		case <-time.After(1 * time.Second):
			t.Error("expected context cancellation")
		}
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	ctx, cancel := context.WithCancel(context.Background())
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "/test", nil)

	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	r.ServeHTTP(w, req)
}
