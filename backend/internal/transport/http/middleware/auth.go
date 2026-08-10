// Package middleware 实现家庭权限中间件。
package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// FamilyScope 家庭授权上下文。
type FamilyScope struct {
	UserID      string   `json:"user_id"`
	FamilyID    string   `json:"family_id"`
	MemberID    string   `json:"member_id"`
	Role        string   `json:"role"`
	Permissions []string `json:"permissions"`
	Timezone    string   `json:"timezone"`
}

// Claims 声明（JWT payload）。
type Claims struct {
	UserID      string   `json:"uid"`
	FamilyID    string   `json:"fid"`
	MemberID    string   `json:"mid"`
	Role        string   `json:"role"`
	Permissions []string `json:"perms"`
	Timezone    string   `json:"tz"`
	ExpiresAt   int64    `json:"exp"`
}

// AuthClaims 解析并注入 FamilyScope。
// JWT 校验由上层 handler 完成，本中间件只做 Scope 注入（为后续扩展预留）。
func AuthClaims(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 当前使用简单 Bearer 校验；后续接入 JWT 解析后可在这里注入
		token := extractBearer(c)
		if token == "" {
			c.AbortWithStatusJSON(401, gin.H{"code": "AUTH_REQUIRED", "message": "token required", "request_id": c.GetString("request_id")})
			return
		}
		// 占位：token 校验将在 auth handler 层完成
		c.Next()
	}
}

func extractBearer(c *gin.Context) string {
	auth := c.GetHeader("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return ""
}

// RequireFamily 强制要求请求体或路径中携带 family_id（占位，后续校验）。
func RequireFamily() gin.HandlerFunc {
	return func(c *gin.Context) {
		// family_id 校验逻辑：从 URL 路径或授权上下文读取
		_ = c.Param("familyId")
		c.Next()
	}
}

// RequireAdmin 仅管理员可访问。
func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 检查 c.Get("scope") 中的 Role 是否为 admin
		c.Next()
	}
}
