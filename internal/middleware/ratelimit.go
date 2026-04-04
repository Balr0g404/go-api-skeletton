package middleware

import (
	"fmt"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

// luaRateLimit atomically increments the counter and sets the TTL on first call,
// preventing the race between INCR and EXPIRE.
var luaRateLimit = redis.NewScript(`
local count = redis.call('INCR', KEYS[1])
if count == 1 then
    redis.call('EXPIRE', KEYS[1], ARGV[1])
end
return count
`)

// memRateLimiter is an in-memory fallback used when Redis is unavailable.
type memRateLimiter struct {
	mu      sync.Mutex
	entries map[string]*memEntry
}

type memEntry struct {
	count  int64
	expiry time.Time
}

func (m *memRateLimiter) increment(key string, window time.Duration) int64 {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	entry, ok := m.entries[key]
	if !ok || now.After(entry.expiry) {
		entry = &memEntry{expiry: now.Add(window)}
		m.entries[key] = entry
	}
	entry.count++
	return entry.count
}

var globalMemLimiter = &memRateLimiter{entries: make(map[string]*memEntry)}

func RateLimit(redisClient *redis.Client, limit int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := fmt.Sprintf("ratelimit:%s", c.ClientIP())

		count, err := luaRateLimit.Run(c.Request.Context(), redisClient, []string{key}, int(window.Seconds())).Int64()
		if err != nil {
			// Redis unavailable: fall back to in-memory limiter.
			log.Warn().Err(err).Str("ip", c.ClientIP()).Msg("redis unavailable for rate limit, using in-memory fallback")
			count = globalMemLimiter.increment(key, window)
		}

		if count > int64(limit) {
			c.Header("Retry-After", fmt.Sprintf("%d", int(window.Seconds())))
			c.JSON(429, gin.H{
				"success": false,
				"error":   "rate limit exceeded",
			})
			c.Abort()
			return
		}

		remaining := int64(limit) - count
		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))

		c.Next()
	}
}
