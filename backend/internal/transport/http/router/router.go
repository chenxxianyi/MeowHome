// Package router 组装 HTTP 路由。
package router

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/meowhome/backend/internal/platform/config"
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

	// 家庭与猫咪
	familyG := protected.Group("/families")
	familyG.POST("", h.CreateFamily)
	familyG.GET("/:familyId", h.GetFamily)
	familyG.PATCH("/:familyId", h.UpdateFamily)
	familyG.GET("/:familyId/members", h.ListMembers)
	familyG.GET("/:familyId/cats", h.ListCats)
	familyG.POST("/:familyId/cats", h.CreateCat)
	familyG.GET("/:familyId/cats/:catId", h.GetCat)
	familyG.PATCH("/:familyId/cats/:catId", h.UpdateCat)
	familyG.DELETE("/:familyId/cats/:catId", h.DeleteCat)
}

func registerV1Placeholder(r *gin.Engine) {
	v1 := r.Group("/api/v1")
	v1.GET("/ping", func(c *gin.Context) {
		response.OK(c, gin.H{"message": "pong"})
	})
}

// AuthMiddleware 校验 Bearer JWT 并注入 user_id。
func AuthMiddleware(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":       "AUTH_REQUIRED",
				"message":    "token required",
				"request_id": c.GetString("request_id"),
			})
			c.Abort()
			return
		}
		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		lines := strings.Split(tokenStr, ".")
		if len(lines) != 3 {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":       "TOKEN_EXPIRED",
				"message":    "invalid token",
				"request_id": c.GetString("request_id"),
			})
			c.Abort()
			return
		}
		userID := lines[0]
		c.Set("user_id", userID)
		c.Next()
	}
}
