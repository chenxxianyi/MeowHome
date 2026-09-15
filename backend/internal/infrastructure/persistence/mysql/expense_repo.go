// Package mysql 实现 Expense 仓储。
package mysql

import (
	"context"

	"gorm.io/gorm"

	"github.com/meowhome/backend/internal/domain/model"
	"github.com/meowhome/backend/internal/domain/repository"
)

// ExpenseRepo 实现 repository.ExpenseRepo。
type ExpenseRepo struct {
	db *gorm.DB
}

// NewExpenseRepo 创建 Expense 仓储。
func NewExpenseRepo(db *gorm.DB) *ExpenseRepo {
	return &ExpenseRepo{db: db}
}

var _ repository.ExpenseRepo = (*ExpenseRepo)(nil)

// Create 创建支出。
func (r *ExpenseRepo) Create(ctx context.Context, e *model.Expense) error {
	return r.db.WithContext(ctx).Create(e).Error
}

// FindByID 按 ID 查找支出。
func (r *ExpenseRepo) FindByID(ctx context.Context, id string) (*model.Expense, error) {
	var e model.Expense
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&e).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}
	return &e, nil
}

// List 按家庭（可选月份 YYYY-MM）查询支出。
// occurred_on 是 ISO 定长字符串，字典序即时间序，可直接做前缀匹配。
func (r *ExpenseRepo) List(ctx context.Context, q repository.ExpenseQuery) ([]*model.Expense, error) {
	tx := r.db.WithContext(ctx).Model(&model.Expense{}).Where("family_id = ?", q.FamilyID)
	if q.Month != "" {
		tx = tx.Where("occurred_on LIKE ?", q.Month+"%")
	}

	var out []*model.Expense
	if err := tx.Order("occurred_on DESC, created_at DESC").Limit(500).Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}
