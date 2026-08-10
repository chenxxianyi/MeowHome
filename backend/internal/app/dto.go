// Package app 提供应用层 DTO 定义。
package app

import (
	"github.com/meowhome/backend/internal/domain/model"
)

// RegisterRequest 注册请求体。
type RegisterRequest struct {
	Email      string `json:"email"`
	Password   string `json:"password"`
	UserName   string `json:"user_name"`
	FamilyName string `json:"family_name,omitempty"`
}

// LoginRequest 登录请求体。
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// RefreshRequest 刷新请求体。
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// RegisterEnvelope 认证结果响应（注册/登录/刷新共用）。
type RegisterEnvelope struct {
	AccessToken  string      `json:"access_token"`
	RefreshToken string      `json:"refresh_token"`
	ExpiresIn    int64       `json:"expires_in"`
	User         *model.User `json:"user"`
}
