// Package mysql 实现 InventoryTransaction 仓储。
package mysql

import (
	"context"

	"gorm.io/gorm"

	"github.com/meowhome/backend/internal/domain/model"
	"github.com/meowhome/backend/internal/domain/repository"
)

// InventoryTransactionRepo 实现 repository.InventoryTransactionRepo。
type InventoryTransactionRepo struct {
	db *gorm.DB
}

// NewInventoryTransactionRepo 创建库存流水仓储。
func NewInventoryTransactionRepo(db *gorm.DB) *InventoryTransactionRepo {
	return &InventoryTransactionRepo{db: db}
}

var _ repository.InventoryTransactionRepo = (*InventoryTransactionRepo)(nil)

// Create 写入一条库存流水。
func (r *InventoryTransactionRepo) Create(ctx context.Context, tx *model.InventoryTransaction) error {
	return r.db.WithContext(ctx).Create(tx).Error
}

// ListByFamily 列出家庭库存流水；itemID 为空表示全部，按 created_at 倒序。
func (r *InventoryTransactionRepo) ListByFamily(ctx context.Context, familyID, itemID string, limit int) ([]*model.InventoryTransaction, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	var out []*model.InventoryTransaction
	query := r.db.WithContext(ctx).Where("family_id = ?", familyID)
	if itemID != "" {
		query = query.Where("item_id = ?", itemID)
	}
	if err := query.Order("created_at DESC, id DESC").Limit(limit).Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}
