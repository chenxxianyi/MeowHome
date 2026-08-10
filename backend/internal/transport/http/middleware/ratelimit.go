// Package middleware 实现基础限流中间件（进程内，Redis 接入后可替换）。
package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/meowhome/backend/internal/platform/errors"
)

// RateLimiter 简单的进程内令牌桶限流器。
type RateLimiter struct {
	mu      sync.Mutex
	clients map[string]*bucket
	limit   int
	window  time.Duration
	logger  *zap.Logger
}

type bucket struct {
	count int
	reset time.Time
}

// NewRateLimiter 创建限流器。
func NewRateLimiter(logger *zap.Logger, limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		clients: make(map[string]*bucket),
		limit:   limit,
		window:  window,
		logger:  logger,
	}
}

// Handle 限流中间件。
func (rl *RateLimiter) Handle(c *gin.Context) {
	key := c.ClientIP()
	if key == "" {
		key = "unknown"
	}
	rl.mu.Lock()
	b, exists := rl.clients[key]
	if !exists || time.Now().After(b.reset) {
		b = &bucket{count: 0, reset: time.Now().Add(rl.window)}
		rl.clients[key] = b
	}
	b.count++
	rl.mu.Unlock()

	if b.count > rl.limit {
		rl.logger.Warn("rate limit exceeded", zap.String("ip", key), zap.Int("count", b.count))
		c.JSON(http.StatusTooManyRequests, gin.H{
			"code":       errors.CodeSuccess,
			"message":    "rate limit exceeded",
			"request_id": c.GetString("request_id"),
		})
		c.Abort()
		return
	}
	c.Header("X-RateLimit-Limit", string(rune(rl.limit)))
	c.Next()
}
