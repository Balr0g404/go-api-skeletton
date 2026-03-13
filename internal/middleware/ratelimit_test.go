package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Balr0g404/go-api-skeletton/internal/middleware"
)

func newRateLimitEngine(t *testing.T, limit int, window time.Duration) (*gin.Engine, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(func() { mr.Close() })

	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	r := gin.New()
	r.Use(middleware.RateLimit(client, limit, window))
	r.GET("/test", func(c *gin.Context) { c.Status(http.StatusOK) })
	return r, mr
}

func doRequest(r *gin.Engine) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	r.ServeHTTP(w, req)
	return w
}

func TestRateLimit_AllowsRequestsBelowLimit(t *testing.T) {
	r, _ := newRateLimitEngine(t, 3, time.Minute)

	for i := 0; i < 3; i++ {
		w := doRequest(r)
		assert.Equal(t, http.StatusOK, w.Code)
	}
}

func TestRateLimit_BlocksWhenLimitExceeded(t *testing.T) {
	r, _ := newRateLimitEngine(t, 2, time.Minute)

	doRequest(r)
	doRequest(r)
	w := doRequest(r)

	assert.Equal(t, http.StatusTooManyRequests, w.Code)
}

func TestRateLimit_SetsRateLimitHeaders(t *testing.T) {
	r, _ := newRateLimitEngine(t, 5, time.Minute)

	w := doRequest(r)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "5", w.Header().Get("X-RateLimit-Limit"))
	assert.Equal(t, "4", w.Header().Get("X-RateLimit-Remaining"))
}

func TestRateLimit_SetsRetryAfterOnBlock(t *testing.T) {
	r, _ := newRateLimitEngine(t, 1, time.Minute)

	doRequest(r)
	w := doRequest(r)

	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.Equal(t, "60", w.Header().Get("Retry-After"))
}

func TestRateLimit_AllowsAfterWindowExpires(t *testing.T) {
	r, mr := newRateLimitEngine(t, 1, time.Second)

	doRequest(r)
	w := doRequest(r)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)

	mr.FastForward(2 * time.Second)

	w = doRequest(r)
	assert.Equal(t, http.StatusOK, w.Code)
}
