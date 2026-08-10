// Package mysql 实现 Cat 仓储。
package mysql

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/meowhome/backend/internal/domain/model"
	"github.com/meowhome/backend/internal/domain/repository"
)

// CatRepo 实现 repository.CatRepo。
type CatRepo struct {
	db *gorm.DB
}

// NewCatRepo 创建 Cat 仓储。
func NewCatRepo(db *gorm.DB) *CatRepo {
	return &CatRepo{db: db}
}

var _ repository.CatRepo = (*CatRepo)(nil)

// Create 创建猫咪。
func (r *CatRepo) Create(ctx context.Context, c *model.Cat) error {
	return r.db.WithContext(ctx).Create(c).Error
}

// FindByID 按 ID 查找猫咪（含软删除）。
func (r *CatRepo) FindByID(ctx context.Context, id string) (*model.Cat, error) {
	var c model.Cat
	if err := r.db.WithContext(ctx).Unscoped().Where("id = ?", id).First(&c).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

// ListByFamily 列出家庭所有猫咪（含软删除以保留历史）。
func (r *CatRepo) ListByFamily(ctx context.Context, familyID string) ([]*model.Cat, error) {
	var cats []*model.Cat
	if err := r.db.WithContext(ctx).Unscoped().Where("family_id = ?", familyID).Find(&cats).Error; err != nil {
		return nil, err
	}
	return cats, nil
}

// Update 更新猫咪。
func (r *CatRepo) Update(ctx context.Context, c *model.Cat) error {
	return r.db.WithContext(ctx).Save(c).Error
}

// Delete 软删除猫咪。
func (r *CatRepo) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Model(&model.Cat{}).Where("id = ?", id).Update("deleted_at", time.Now().UTC()).Error
}
