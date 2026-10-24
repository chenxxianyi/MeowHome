// Package mysql 实现 Inventory 仓储。
package mysql

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

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

// Adjust 在同一事务内锁定库存项、更新数量并写入流水。
//
// 行锁（SELECT ... FOR UPDATE）保证并发调整不会丢失更新；流水插入失败时
// 数量更新一并回滚，避免“有数量没流水”或相反。
func (r *InventoryRepo) Adjust(ctx context.Context, familyID, itemID string, delta int, tx *model.InventoryTransaction) (*model.InventoryItem, error) {
	var updated *model.InventoryItem
	err := r.db.WithContext(ctx).Transaction(func(db *gorm.DB) error {
		var item model.InventoryItem
		if err := db.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND family_id = ? AND deleted_at IS NULL", itemID, familyID).
			First(&item).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return repository.ErrNotFound
			}
			return err
		}
		next := item.Quantity + delta
		if next < 0 {
			// 数量不得为负：按冲突处理，由服务层转成明确错误。
			return repository.ErrConflict
		}
		if err := db.Model(&model.InventoryItem{}).
			Where("id = ? AND family_id = ?", itemID, familyID).
			Update("quantity", next).Error; err != nil {
			return err
		}
		tx.FamilyID = familyID
		tx.ItemID = itemID
		if err := db.Create(tx).Error; err != nil {
			return err
		}
		item.Quantity = next
		updated = &item
		return nil
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}
