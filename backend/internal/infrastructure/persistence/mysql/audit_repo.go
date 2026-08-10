// Package mysql 实现 AuditLog 仓储。
package mysql

import (
	"context"

	"gorm.io/gorm"

	"github.com/meowhome/backend/internal/domain/model"
)

// AuditRepo 实现 repository.AuditRepo。
type AuditRepo struct {
	db *gorm.DB
}

// NewAuditRepo 创建 Audit 仓储。
func NewAuditRepo(db *gorm.DB) *AuditRepo {
	return &AuditRepo{db: db}
}

// Create 写入审计日志。
func (r *AuditRepo) Create(ctx context.Context, a *model.AuditLog) error {
	return r.db.WithContext(ctx).Create(a).Error
}
