// Package mysql 实现 DailyRecord 仓储。
package mysql

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/meowhome/backend/internal/domain/model"
	"github.com/meowhome/backend/internal/domain/repository"
)

// DailyRecordRepo 实现 repository.DailyRecordRepo。
type DailyRecordRepo struct {
	db *gorm.DB
}

// NewDailyRecordRepo 创建 DailyRecord 仓储。
func NewDailyRecordRepo(db *gorm.DB) *DailyRecordRepo {
	return &DailyRecordRepo{db: db}
}

var _ repository.DailyRecordRepo = (*DailyRecordRepo)(nil)

// Create 创建记录。
func (r *DailyRecordRepo) Create(ctx context.Context, rec *model.DailyRecord) error {
	return r.db.WithContext(ctx).Create(rec).Error
}

// FindByID 按 ID 查找。
func (r *DailyRecordRepo) FindByID(ctx context.Context, id string) (*model.DailyRecord, error) {
	var rec model.DailyRecord
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&rec).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}
	return &rec, nil
}

// ListByFamily 按家庭查询记录。
// CatID 过滤走 JSON_CONTAINS：cat_ids 是存 JSON 数组的 TEXT 列。
func (r *DailyRecordRepo) ListByFamily(ctx context.Context, q repository.DailyRecordQuery) ([]*model.DailyRecord, error) {
	tx := r.db.WithContext(ctx).Model(&model.DailyRecord{}).Where("family_id = ?", q.FamilyID)

	if q.CatID != "" {
		tx = tx.Where("cat_ids IS NOT NULL AND JSON_VALID(cat_ids) AND JSON_CONTAINS(cat_ids, JSON_QUOTE(?))", q.CatID)
	}
	if len(q.Types) > 0 {
		tx = tx.Where("record_type IN ?", q.Types)
	}
	if q.From != "" {
		if t, err := parseTimeBound(q.From, false); err == nil {
			tx = tx.Where("occurred_at >= ?", t)
		}
	}
	if q.To != "" {
		if t, err := parseTimeBound(q.To, true); err == nil {
			tx = tx.Where("occurred_at <= ?", t)
		}
	}

	limit := q.Limit
	if limit <= 0 || limit > 1000 {
		limit = 500
	}

	var out []*model.DailyRecord
	if err := tx.Order("occurred_at DESC").Limit(limit).Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

// Delete 软删除记录。
func (r *DailyRecordRepo) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Model(&model.DailyRecord{}).
		Where("id = ?", id).Update("deleted_at", time.Now().UTC()).Error
}

// parseTimeBound 解析查询边界：接受 RFC3339 或 YYYY-MM-DD。
// endOfDay 为真时把纯日期推进到当天 23:59:59.999。
func parseTimeBound(s string, endOfDay bool) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return time.Time{}, err
	}
	if endOfDay {
		t = t.Add(24*time.Hour - time.Millisecond)
	}
	return t, nil
}
