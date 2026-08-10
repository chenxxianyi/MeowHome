// Package app 提供应用层业务逻辑（认证、家庭、猫咪）。
package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/meowhome/backend/internal/domain/model"
	"github.com/meowhome/backend/internal/domain/repository"
	apperr "github.com/meowhome/backend/internal/platform/errors"
)

// AuthService 认证服务。
type AuthService struct {
	userRepo     repository.UserRepo
	refreshToken repository.RefreshTokenRepo
	clock        func() time.Time
	secret       []byte
	accessTTL    time.Duration
}

// NewAuthService 创建认证服务。
func NewAuthService(userRepo repository.UserRepo, refreshToken repository.RefreshTokenRepo, secret []byte, accessTTL time.Duration) *AuthService {
	return &AuthService{
		userRepo:     userRepo,
		refreshToken: refreshToken,
		clock:        time.Now,
		secret:       secret,
		accessTTL:    accessTTL,
	}
}

// AuthResult 认证结果。
type AuthResult struct {
	AccessToken  string      `json:"access_token"`
	RefreshToken string      `json:"refresh_token"`
	ExpiresIn    int64       `json:"expires_in"` // 秒
	User         *model.User `json:"user"`
}

// Register 注册新用户（首次注册必须携带家庭名，B3 简化版先仅建用户）。
func (s *AuthService) Register(ctx context.Context, email, password, userName string) (*AuthResult, error) {
	if email == "" || password == "" || userName == "" {
		return nil, apperr.InvalidRequest(apperr.CodeValidationFailed, "email, password, user_name are required")
	}
	if len(password) < 8 {
		return nil, apperr.InvalidRequest(apperr.CodeValidationFailed, "password must be at least 8 characters")
	}

	now := time.Now().UTC()
	user := &model.User{
		Base: model.Base{
			ID:        model.NewBase("", "").ID,
			CreatedBy: "system",
			CreatedAt: now,
			UpdatedAt: now,
		},
		Email:    email,
		Name:     userName,
		Password: HashPassword(password),
	}
	if err := s.userRepo.Create(ctx, user); err != nil {
		if isDuplicate(err) {
			return nil, apperr.Conflict(apperr.CodeConflict, "email already registered")
		}
		return nil, apperr.Wrap(apperr.TypeInternal, apperr.CodeInvalidRequest, "failed to create user", err)
	}

	refreshTokenRaw, err := s.newRefreshToken(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	accessToken, err := s.newAccessToken(user)
	if err != nil {
		return nil, err
	}

	return &AuthResult{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshTokenRaw,
		ExpiresIn:    int64(s.accessTTL.Seconds()),
	}, nil
}

// Login 登录，成功返回令牌对。
func (s *AuthService) Login(ctx context.Context, email, password string) (*AuthResult, error) {
	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		return nil, apperr.Unauthorized(apperr.CodeAuthRequired, "invalid credentials")
	}
	if !checkPassword(user.Password, password) {
		return nil, apperr.Unauthorized(apperr.CodeAuthRequired, "invalid credentials")
	}

	refreshTokenRaw, err := s.newRefreshToken(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	accessToken, err := s.newAccessToken(user)
	if err != nil {
		return nil, err
	}

	return &AuthResult{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshTokenRaw,
		ExpiresIn:    int64(s.accessTTL.Seconds()),
	}, nil
}

// Refresh 通过 refresh token 换取新令牌对。
func (s *AuthService) Refresh(ctx context.Context, refreshTokenRaw string) (*AuthResult, error) {
	hash := hashToken(refreshTokenRaw)
	rt, err := s.refreshToken.FindByHash(ctx, hash)
	if err != nil {
		return nil, apperr.Unauthorized(apperr.CodeTokenExpired, "invalid refresh token")
	}
	if rt.RevokedAt != nil {
		return nil, apperr.Unauthorized(apperr.CodeTokenExpired, "token revoked")
	}
	if s.clock().UTC().After(rt.ExpiresAt) {
		return nil, apperr.Unauthorized(apperr.CodeTokenExpired, "token expired")
	}

	// 撤销旧 token，签发新 token（旋转）
	_ = s.refreshToken.Revoke(ctx, rt.ID)
	refreshTokenRaw, err = s.newRefreshToken(ctx, rt.UserID)
	if err != nil {
		return nil, err
	}

	user, err := s.userRepo.FindByID(ctx, rt.UserID)
	if err != nil {
		return nil, apperr.Unauthorized(apperr.CodeAuthRequired, "user not found")
	}
	accessToken, err := s.newAccessToken(user)
	if err != nil {
		return nil, err
	}

	return &AuthResult{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshTokenRaw,
		ExpiresIn:    int64(s.accessTTL.Seconds()),
	}, nil
}

// Logout 注销当前 refresh token。
func (s *AuthService) Logout(ctx context.Context, refreshTokenRaw string) error {
	hash := hashToken(refreshTokenRaw)
	rt, err := s.refreshToken.FindByHash(ctx, hash)
	if err != nil {
		// 不存在的 token 视为已登出
		return nil
	}
	return s.refreshToken.Revoke(ctx, rt.ID)
}

// newRefreshToken 生成并持久化一条 refresh token。
func (s *AuthService) newRefreshToken(ctx context.Context, userID string) (string, error) {
	raw := randomToken()
	now := time.Now().UTC()
	rt := &model.RefreshToken{
		Base: model.Base{
			ID:        model.NewBase("", "").ID,
			CreatedBy: userID,
			CreatedAt: now,
			UpdatedAt: now,
		},
		UserID:    userID,
		TokenHash: hashToken(raw),
		IssuedAt:  now,
		ExpiresAt: now.Add(30 * 24 * time.Hour), // 30 天
	}
	if err := s.refreshToken.Create(ctx, rt); err != nil {
		return "", apperr.Wrap(apperr.TypeInternal, apperr.CodeInvalidRequest, "failed to persist refresh token", err)
	}
	return raw, nil
}

// newAccessToken 生成 JWT（B3 简化版：HMAC 签名带前缀）。
func (s *AuthService) newAccessToken(user *model.User) (string, error) {
	now := s.clock().UTC().Unix()
	exp := now + int64(s.accessTTL.Seconds())
	payload := map[string]any{
		"uid": user.ID,
		"exp": exp,
		"iat": now,
	}

	body := hex.EncodeToString(sha256.New().Sum([]byte(structToString(payload))))
	unsigned := user.ID + "." + body
	return unsigned + "." + hex.EncodeToString(hmacSign(s.secret, []byte(unsigned))), nil
}

func isDuplicate(err error) bool {
	return errors.Is(err, repository.ErrDuplicateKey)
}

func hashToken(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}
