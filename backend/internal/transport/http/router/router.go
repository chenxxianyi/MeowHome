// Package router 组装 HTTP 路由。
package router

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/meowhome/backend/internal/platform/config"
	apperr "github.com/meowhome/backend/internal/platform/errors"
	"github.com/meowhome/backend/internal/platform/token"
	"github.com/meowhome/backend/internal/transport/http/handler"
	my "github.com/meowhome/backend/internal/transport/http/middleware"
	"github.com/meowhome/backend/internal/transport/http/response"
)

// HealthChecker 就绪/存活检查。
type HealthChecker interface {
	Liveness() (any, error)
	Readiness() (any, error)
}

// New 构建 Gin 引擎。
func New(
	cfg *config.Config,
	logger *zap.Logger,
	limiter *my.RateLimiter,
	health HealthChecker,
	h *handler.Handler,
) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(my.Recover(logger))
	r.Use(my.NoCache())

	logCtx := &my.RequestContext{Logger: logger}
	r.Use(logCtx.Handle(func(c *gin.Context) {}))
	r.Use(my.CORS(cfg.App.AllowedOrigins))

	registerHealth(r, health)

	if h != nil {
		registerV1(r, h, cfg.Auth.JWTSecret)
	} else {
		registerV1Placeholder(r)
	}
	_ = limiter

	return r
}

func registerHealth(r *gin.Engine, health HealthChecker) {
	r.GET("/health/live", func(c *gin.Context) {
		data, err := health.Liveness()
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "down"})
			return
		}
		response.OK(c, data)
	})
	r.GET("/health/ready", func(c *gin.Context) {
		data, err := health.Readiness()
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "down"})
			return
		}
		response.OK(c, data)
	})
}

func registerV1(r *gin.Engine, h *handler.Handler, jwtSecret string) {
	v1 := r.Group("/api/v1")
	v1.GET("/ping", func(c *gin.Context) {
		response.OK(c, gin.H{"message": "pong"})
	})

	// 认证（无需登录）
	authG := v1.Group("/auth")
	authG.POST("/register", h.Register)
	authG.POST("/login", h.Login)
	authG.POST("/refresh", h.Refresh)
	authG.POST("/logout", h.Logout)

	// 需要认证的路由
	protected := v1.Group("")
	protected.Use(AuthMiddleware(jwtSecret))
	protected.GET("/me", h.Me)
	protected.GET("/me/families", h.ListMyFamilies)

	// 家庭与猫咪
	familyG := protected.Group("/families")
	familyG.GET("", h.ListMyFamilies)
	familyG.POST("", h.CreateFamily)
	familyG.GET("/:familyId", h.GetFamily)
	familyG.PATCH("/:familyId", h.UpdateFamily)
	familyG.GET("/:familyId/members", h.ListMembers)
	familyG.GET("/:familyId/cats", h.ListCats)
	familyG.POST("/:familyId/cats", h.CreateCat)
	familyG.GET("/:familyId/cats/:catId", h.GetCat)
	familyG.PATCH("/:familyId/cats/:catId", h.UpdateCat)
	familyG.DELETE("/:familyId/cats/:catId", h.DeleteCat)

	// 日常记录
	familyG.GET("/:familyId/records", h.ListRecords)
	familyG.POST("/:familyId/records", h.CreateRecord)
	familyG.POST("/:familyId/records/batch", h.CreateRecordBatch)

	// 聚合视图
	familyG.GET("/:familyId/today", h.TodayStatus)
	familyG.GET("/:familyId/focus", h.FocusItems)
	familyG.GET("/:familyId/trends", h.Trends)

	// 提醒
	familyG.GET("/:familyId/reminders", h.ListReminders)
	familyG.POST("/:familyId/reminders", h.CreateReminder)
	familyG.PATCH("/:familyId/reminders/:reminderId/complete", h.CompleteReminder)

	// 时光
	familyG.GET("/:familyId/moments", h.ListMoments)
	familyG.POST("/:familyId/moments", h.CreateMoment)

	// 库存
	familyG.GET("/:familyId/inventory", h.ListInventory)
	familyG.POST("/:familyId/inventory", h.CreateInventoryItem)

	// 账目
	familyG.GET("/:familyId/expenses", h.ListExpenses)
	familyG.POST("/:familyId/expenses", h.CreateExpense)

	// AI
	familyG.POST("/:familyId/ai/parse", h.ParseAI)
	familyG.GET("/:familyId/ai/summary", h.AISummary)
}

func registerV1Placeholder(r *gin.Engine) {
	v1 := r.Group("/api/v1")
	v1.GET("/ping", func(c *gin.Context) {
		response.OK(c, gin.H{"message": "pong"})
	})
}

// AuthMiddleware 校验 Bearer 访问令牌的 HMAC 签名与过期时间，并注入 user_id。
// 缺失/篡改的令牌返回 AUTH_REQUIRED，过期返回 TOKEN_EXPIRED。
func AuthMiddleware(secret string) gin.HandlerFunc {
	key := []byte(secret)
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			abortUnauthorized(c, apperr.CodeAuthRequired, "token required")
			return
		}
		raw := strings.TrimPrefix(authHeader, "Bearer ")

		claims, err := token.Parse(key, raw, time.Now().UTC())
		if err != nil {
			if errors.Is(err, token.ErrExpired) {
				abortUnauthorized(c, apperr.CodeTokenExpired, "token expired")
				return
			}
			abortUnauthorized(c, apperr.CodeAuthRequired, "invalid token")
			return
		}

		c.Set("user_id", claims.UserID)
		c.Next()
	}
}

func abortUnauthorized(c *gin.Context, code, message string) {
	c.JSON(http.StatusUnauthorized, gin.H{
		"code":       code,
		"message":    message,
		"request_id": c.GetString("request_id"),
	})
	c.Abort()
}
