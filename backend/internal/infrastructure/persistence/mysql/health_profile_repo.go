// Package mysql 实现 CatHealthProfile 仓储。
package mysql

import (
	"context"

	"gorm.io/gorm"

	"github.com/meowhome/backend/internal/domain/model"
)

// HealthProfileRepo 实现 repository.HealthRepo（健康档案子集）。
type HealthProfileRepo struct {
	db *gorm.DB
}

// NewHealthProfileRepo 创建健康档案仓储。
func NewHealthProfileRepo(db *gorm.DB) *HealthProfileRepo {
	return &HealthProfileRepo{db: db}
}

// Create 创建健康档案。
func (r *HealthProfileRepo) Create(ctx context.Context, p *model.CatHealthProfile) error {
	return r.db.WithContext(ctx).Create(p).Error
}

// FindByCatID 按猫咪 ID 查找健康档案（含软删除）。
func (r *HealthProfileRepo) FindByCatID(ctx context.Context, catID string) (*model.CatHealthProfile, error) {
	var p model.CatHealthProfile
	if err := r.db.WithContext(ctx).Unscoped().Where("cat_id = ?", catID).First(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

// Update 更新健康档案。
func (r *HealthProfileRepo) Update(ctx context.Context, p *model.CatHealthProfile) error {
	return r.db.WithContext(ctx).Save(p).Error
}
