package middleware

import (
	"os"
	"strconv"

	"github.com/gin-gonic/gin"

	"short-video-platform/api-gateway/internal/response"
	"short-video-platform/pkg/redislimit"
)

// RateLimit applies per-IP limits via Redis when available (no global shared bucket).
func RateLimit() gin.HandlerFunc {
	redisLimiter, _ := redislimit.NewFromEnv()

	return func(c *gin.Context) {
		if redisLimiter == nil {
			c.Next()
			return
		}
		key := redislimit.ClientIPKey(c.ClientIP())
		ok, err := redisLimiter.Allow(c.Request.Context(), key)
		if err == nil && !ok {
			response.Fail(c, 429, 42900, "too many requests")
			c.Abort()
			return
		}
		c.Next()
	}
}

func parseFloatEnv(key string, def float64) float64 {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil || f <= 0 {
		return def
	}
	return f
}

func parseIntEnv(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return def
	}
	return n
}
