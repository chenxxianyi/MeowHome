// Package mysql 实现 ShoppingList 仓储。
package mysql

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/meowhome/backend/internal/domain/model"
	"github.com/meowhome/backend/internal/domain/repository"
)

// ShoppingListRepo 实现 repository.ShoppingListRepo。
type ShoppingListRepo struct {
	db *gorm.DB
}

// NewShoppingListRepo 创建待购买项仓储。
func NewShoppingListRepo(db *gorm.DB) *ShoppingListRepo {
	return &ShoppingListRepo{db: db}
}

var _ repository.ShoppingListRepo = (*ShoppingListRepo)(nil)

// Create 创建待购买项。
func (r *ShoppingListRepo) Create(ctx context.Context, item *model.ShoppingListItem) error {
	return r.db.WithContext(ctx).Create(item).Error
}

// FindByID 按家庭与 ID 查找待购买项。
func (r *ShoppingListRepo) FindByID(ctx context.Context, familyID, id string) (*model.ShoppingListItem, error) {
	var item model.ShoppingListItem
	if err := r.db.WithContext(ctx).
		Where("id = ? AND family_id = ? AND deleted_at IS NULL", id, familyID).
		First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}
	return &item, nil
}

// ListByFamily 按家庭列出待购买项；status 为空表示全部。
// 未完成项优先展示，其余按创建时间倒序。
func (r *ShoppingListRepo) ListByFamily(ctx context.Context, familyID, status string) ([]*model.ShoppingListItem, error) {
	var out []*model.ShoppingListItem
	query := r.db.WithContext(ctx).Where("family_id = ? AND deleted_at IS NULL", familyID)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if err := query.Order("CASE WHEN status = 'todo' THEN 0 ELSE 1 END ASC, created_at DESC").
		Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

// Update 更新待购买项（调用方已通过 FindByID 校验家庭归属）。
func (r *ShoppingListRepo) Update(ctx context.Context, item *model.ShoppingListItem) error {
	item.UpdatedAt = time.Now().UTC()
	return r.db.WithContext(ctx).Save(item).Error
}

// Delete 软删除待购买项。
func (r *ShoppingListRepo) Delete(ctx context.Context, familyID, id string) error {
	return r.db.WithContext(ctx).Model(&model.ShoppingListItem{}).
		Where("id = ? AND family_id = ? AND deleted_at IS NULL", id, familyID).
		Update("deleted_at", time.Now().UTC()).Error
}
