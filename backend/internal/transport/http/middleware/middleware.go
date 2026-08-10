// Package middleware 定义 HTTP 中间件。
package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/meowhome/backend/internal/platform/errors"
)

// RequestContext 写入请求上下文。
type RequestContext struct {
	Logger *zap.Logger
	Next   func()
}

// RequestID 生成并注入请求 ID，附带日志字段。
func (r *RequestContext) Handle(next gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := generateRequestID(c)
		c.Set("request_id", id)
		c.Header("X-Request-Id", id)

		start := time.Now()
		next(c)

		fields := []zap.Field{
			zap.String("request_id", id),
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", time.Since(start)),
		}
		if clientIP := c.ClientIP(); clientIP != "" {
			fields = append(fields, zap.String("client_ip", clientIP))
		}
		r.Logger.Info("request", fields...)
	}
}

func generateRequestID(c *gin.Context) string {
	if id := c.GetHeader("X-Request-Id"); id != "" {
		return id
	}
	b := make([]byte, 8)
	if _, err := rand.Read(b); err == nil {
		return hex.EncodeToString(b)
	}
	return uuid.New().String()
}

// Recover 全局恢复中间件：记录 panic，返回 500。
func Recover(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				stack := fmt.Sprintf("%v", err)
				logger.Error("panic recovered",
					zap.Any("error", err),
					zap.String("stack", stack),
					zap.String("request_id", c.GetString("request_id")),
				)
				c.JSON(http.StatusInternalServerError, gin.H{
					"code":       errors.CodeSuccess,
					"message":    "internal error",
					"request_id": c.GetString("request_id"),
				})
				c.Abort()
			}
		}()
		c.Next()
	}
}

// NoCache 禁止缓存动态页面。
func NoCache() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
		c.Header("Pragma", "no-cache")
		c.Header("Expires", "0")
		c.Next()
	}
}

// CORS 基础跨域中间件（实际部署由反向代理处理）。
func CORS(allowedOrigins []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin == "" {
			c.Next()
			return
		}
		for _, o := range allowedOrigins {
			if strings.EqualFold(origin, o) {
				c.Header("Access-Control-Allow-Origin", o)
				c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, Idempotency-Key, X-Request-Id")
				c.Header("Access-Control-Allow-Credentials", "true")
				c.Header("Access-Control-Max-Age", "86400")
				break
			}
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
