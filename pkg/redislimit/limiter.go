package redislimit

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// Limiter 分布式限流（固定窗口计数）。
type Limiter struct {
	client *redis.Client
	prefix string
	limit  int64
	window time.Duration
}

func NewFromEnv() (*Limiter, error) {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		return nil, nil
	}
	limit := int64(parseIntEnv("RATE_LIMIT_RPS", 50))
	if burst := parseIntEnv("RATE_LIMIT_BURST", 100); int64(burst) > limit {
		limit = int64(burst)
	}
	client := redis.NewClient(&redis.Options{Addr: addr})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}
	return &Limiter{
		client: client,
		prefix: getenv("REDIS_RATE_PREFIX", "svp:rl:"),
		limit:  limit,
		window: time.Second,
	}, nil
}

func (l *Limiter) Allow(ctx context.Context, key string) (bool, error) {
	if l == nil {
		return true, nil
	}
	redisKey := l.prefix + key
	pipe := l.client.TxPipeline()
	incr := pipe.Incr(ctx, redisKey)
	pipe.Expire(ctx, redisKey, l.window)
	if _, err := pipe.Exec(ctx); err != nil {
		return false, err
	}
	count, err := incr.Result()
	if err != nil {
		return false, err
	}
	return count <= l.limit, nil
}

func (l *Limiter) Close() error {
	if l == nil {
		return nil
	}
	return l.client.Close()
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
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

func ClientIPKey(ip string) string {
	return fmt.Sprintf("ip:%s", ip)
}
