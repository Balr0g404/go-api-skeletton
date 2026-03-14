package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type timeoutWriter struct {
	gin.ResponseWriter
	mu       sync.Mutex
	timedOut bool
	wrote    bool
}

func (tw *timeoutWriter) WriteHeader(code int) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	if !tw.timedOut {
		tw.wrote = true
		tw.ResponseWriter.WriteHeader(code)
	}
}

func (tw *timeoutWriter) Write(b []byte) (int, error) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	if tw.timedOut {
		return 0, nil
	}
	tw.wrote = true
	return tw.ResponseWriter.Write(b)
}

func (tw *timeoutWriter) WriteString(s string) (int, error) {
	return tw.Write([]byte(s))
}

func (tw *timeoutWriter) sendTimeout() bool {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	if tw.wrote {
		return false
	}
	tw.timedOut = true
	tw.ResponseWriter.WriteHeader(http.StatusServiceUnavailable)
	body, _ := json.Marshal(gin.H{"success": false, "error": "request timeout"})
	_, _ = tw.ResponseWriter.Write(body)
	return true
}

func Timeout(duration time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), duration)
		defer cancel()

		tw := &timeoutWriter{ResponseWriter: c.Writer}
		c.Writer = tw
		c.Request = c.Request.WithContext(ctx)

		done := make(chan struct{})

		go func() {
			c.Next()
			close(done)
		}()

		select {
		case <-done:
		case <-ctx.Done():
			tw.sendTimeout()
			c.Abort()
		}
	}
}
