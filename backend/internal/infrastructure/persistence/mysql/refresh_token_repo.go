// Package mysql 实现 RefreshToken 仓储。
package mysql

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/meowhome/backend/internal/domain/model"
)

// RefreshTokenRepo 管理 refresh token。
type RefreshTokenRepo struct {
	db *gorm.DB
}

// NewRefreshTokenRepo 创建 RefreshToken 仓储。
func NewRefreshTokenRepo(db *gorm.DB) *RefreshTokenRepo {
	return &RefreshTokenRepo{db: db}
}

// Create 创建 refresh token。
func (r *RefreshTokenRepo) Create(ctx context.Context, t *model.RefreshToken) error {
	return r.db.WithContext(ctx).Create(t).Error
}

// FindByHash 按 token hash 查找（含已撤销）。
func (r *RefreshTokenRepo) FindByHash(ctx context.Context, hash string) (*model.RefreshToken, error) {
	var t model.RefreshToken
	if err := r.db.WithContext(ctx).Where("token_hash = ?", hash).First(&t).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

// Revoke 撤销指定 token。
func (r *RefreshTokenRepo) Revoke(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Model(&model.RefreshToken{}).
		Where("id = ?", id).
		Update("revoked_at", time.Now().UTC()).Error
}

// RevokeAllForUser 撤销某用户全部 token。
func (r *RefreshTokenRepo) RevokeAllForUser(ctx context.Context, userID string) error {
	return r.db.WithContext(ctx).Model(&model.RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", time.Now().UTC()).Error
}
