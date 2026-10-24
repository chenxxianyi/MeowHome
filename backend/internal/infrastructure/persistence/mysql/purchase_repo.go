// Package mysql 实现 PurchaseImport 仓储。
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

// PurchaseImportRepo 实现 repository.PurchaseImportRepo。
type PurchaseImportRepo struct {
	db *gorm.DB
}

// NewPurchaseImportRepo 创建采购导入仓储。
func NewPurchaseImportRepo(db *gorm.DB) *PurchaseImportRepo {
	return &PurchaseImportRepo{db: db}
}

var _ repository.PurchaseImportRepo = (*PurchaseImportRepo)(nil)

// Create 创建采购导入记录。
func (r *PurchaseImportRepo) Create(ctx context.Context, imp *model.PurchaseImport) error {
	return r.db.WithContext(ctx).Create(imp).Error
}

// FindByID 按家庭与 ID 查找采购导入。
func (r *PurchaseImportRepo) FindByID(ctx context.Context, familyID, id string) (*model.PurchaseImport, error) {
	var imp model.PurchaseImport
	if err := r.db.WithContext(ctx).
		Where("id = ? AND family_id = ? AND deleted_at IS NULL", id, familyID).
		First(&imp).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}
	return &imp, nil
}

// ListByFamily 列出家庭采购导入历史，按创建时间倒序。
func (r *PurchaseImportRepo) ListByFamily(ctx context.Context, familyID string, limit int) ([]*model.PurchaseImport, error) {
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	var out []*model.PurchaseImport
	if err := r.db.WithContext(ctx).
		Where("family_id = ? AND deleted_at IS NULL", familyID).
		Order("created_at DESC").Limit(limit).Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

// ListLines 列出导入的全部商品行，按行号升序。
func (r *PurchaseImportRepo) ListLines(ctx context.Context, familyID, importID string) ([]*model.PurchaseImportLine, error) {
	var out []*model.PurchaseImportLine
	if err := r.db.WithContext(ctx).
		Where("family_id = ? AND import_id = ? AND deleted_at IS NULL", familyID, importID).
		Order("line_no ASC, created_at ASC").Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

// UpdateImport 更新采购导入记录（调用方已校验家庭归属）。
func (r *PurchaseImportRepo) UpdateImport(ctx context.Context, imp *model.PurchaseImport) error {
	imp.UpdatedAt = time.Now().UTC()
	return r.db.WithContext(ctx).Save(imp).Error
}

// ReplaceLines 在一个事务内替换导入的全部商品行。
// 草稿是用户修订结果，旧行直接删除（Unscoped）而不是软删除，避免草稿堆积。
func (r *PurchaseImportRepo) ReplaceLines(ctx context.Context, familyID, importID string, lines []*model.PurchaseImportLine) error {
	return r.db.WithContext(ctx).Transaction(func(db *gorm.DB) error {
		if err := db.Unscoped().
			Where("family_id = ? AND import_id = ?", familyID, importID).
			Delete(&model.PurchaseImportLine{}).Error; err != nil {
			return err
		}
		if len(lines) == 0 {
			return nil
		}
		for _, line := range lines {
			line.FamilyID = familyID
			line.ImportID = importID
			if err := db.Create(line).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// Confirm 在单事务内完成采购确认落账。
//
// 步骤：锁定导入行（FOR UPDATE，串行化并发确认）→ 幂等短路（已确认直接返回）
// → 逐行建项/加流水/加数量 → 完成待购项 → 写订单支出 → 标记 confirmed。
// 唯一索引（expenses / inventory_transactions 的 family+source）作为第二道防线：
// 即便并发穿过行锁，重复写入也会撞唯一键并整事务回滚。
func (r *PurchaseImportRepo) Confirm(ctx context.Context, familyID, userID, importID string, plan repository.PurchaseConfirmPlan) (*model.PurchaseImport, bool, error) {
	var imp model.PurchaseImport
	created := false
	err := r.db.WithContext(ctx).Transaction(func(db *gorm.DB) error {
		if err := db.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND family_id = ? AND deleted_at IS NULL", importID, familyID).
			First(&imp).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return repository.ErrNotFound
			}
			return err
		}
		if imp.Status == model.PurchaseStatusConfirmed || imp.ConfirmedAt != nil {
			// 幂等：已确认的导入重复提交直接返回当前状态，不产生新落账。
			return nil
		}
		if imp.Status == model.PurchaseStatusCancelled {
			return repository.ErrConflict
		}

		for i := range plan.StockLines {
			line := &plan.StockLines[i]
			if line.NewItem != nil {
				line.NewItem.FamilyID = familyID
				if line.NewItem.CreatedBy == "" {
					line.NewItem.CreatedBy = userID
				}
				if err := db.Create(line.NewItem).Error; err != nil {
					return err
				}
			}
			if line.Tx == nil {
				continue
			}
			targetID := line.Tx.ItemID
			if line.NewItem != nil {
				targetID = line.NewItem.ID
			}
			if targetID == "" {
				return errors.New("purchase confirm: stock line without target item")
			}
			line.Tx.FamilyID = familyID
			line.Tx.ItemID = targetID
			line.Tx.CreatedBy = userID
			if err := db.Create(line.Tx).Error; err != nil {
				if isDuplicate(err) {
					return repository.ErrDuplicateKey
				}
				return err
			}
			if err := db.Model(&model.InventoryItem{}).
				Where("id = ? AND family_id = ?", targetID, familyID).
				Update("quantity", gorm.Expr("quantity + ?", line.Tx.Change)).Error; err != nil {
				return err
			}
		}

		for _, item := range plan.ShoppingCompletions {
			if err := db.Model(&model.ShoppingListItem{}).
				Where("id = ? AND family_id = ? AND deleted_at IS NULL", item.ID, familyID).
				Updates(map[string]any{
					"status":                    model.ShoppingStatusPurchased,
					"completed_at":              item.CompletedAt,
					"matched_inventory_item_id": item.MatchedInventoryItemID,
				}).Error; err != nil {
				return err
			}
		}

		if plan.Expense != nil {
			plan.Expense.FamilyID = familyID
			plan.Expense.SourceType = model.TxReasonPurchaseImport
			plan.Expense.SourceID = importID
			if err := db.Create(plan.Expense).Error; err != nil {
				if isDuplicate(err) {
					return repository.ErrDuplicateKey
				}
				return err
			}
			imp.ExpenseID = plan.Expense.ID
		}

		now := time.Now().UTC()
		imp.Status = model.PurchaseStatusConfirmed
		imp.ConfirmedAt = &now
		imp.UpdatedAt = now
		if err := db.Save(&imp).Error; err != nil {
			return err
		}
		created = true
		return nil
	})
	if err != nil {
		return nil, false, err
	}
	return &imp, created, nil
}
