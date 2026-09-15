// Package mysql 实现 Inventory 仓储。
package mysql

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/meowhome/backend/internal/domain/model"
	"github.com/meowhome/backend/internal/domain/repository"
)

// InventoryRepo 实现 repository.InventoryRepo。
type InventoryRepo struct {
	db *gorm.DB
}

// NewInventoryRepo 创建 Inventory 仓储。
func NewInventoryRepo(db *gorm.DB) *InventoryRepo {
	return &InventoryRepo{db: db}
}

var _ repository.InventoryRepo = (*InventoryRepo)(nil)

// Create 创建库存项。
func (r *InventoryRepo) Create(ctx context.Context, i *model.InventoryItem) error {
	return r.db.WithContext(ctx).Create(i).Error
}

// FindByID 按 ID 查找库存项。
func (r *InventoryRepo) FindByID(ctx context.Context, id string) (*model.InventoryItem, error) {
	var item model.InventoryItem
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&item).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}
	return &item, nil
}

// ListByFamily 列出家庭全部库存项。
func (r *InventoryRepo) ListByFamily(ctx context.Context, familyID string) ([]*model.InventoryItem, error) {
	var out []*model.InventoryItem
	if err := r.db.WithContext(ctx).Where("family_id = ?", familyID).
		Order("category ASC, name ASC").Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

// Update 更新库存项。
func (r *InventoryRepo) Update(ctx context.Context, i *model.InventoryItem) error {
	return r.db.WithContext(ctx).Save(i).Error
}

// Delete 软删除库存项。
func (r *InventoryRepo) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Model(&model.InventoryItem{}).
		Where("id = ?", id).Update("deleted_at", time.Now().UTC()).Error
}
