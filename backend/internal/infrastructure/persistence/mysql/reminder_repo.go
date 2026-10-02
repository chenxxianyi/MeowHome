// Package mysql 实现 Reminder 仓储。
package mysql

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/meowhome/backend/internal/domain/model"
	"github.com/meowhome/backend/internal/domain/repository"
)

// ReminderRepo 实现 repository.ReminderRepo。
type ReminderRepo struct {
	db *gorm.DB
}

// NewReminderRepo 创建 Reminder 仓储。
func NewReminderRepo(db *gorm.DB) *ReminderRepo {
	return &ReminderRepo{db: db}
}

var _ repository.ReminderRepo = (*ReminderRepo)(nil)

// Create 创建提醒。
func (r *ReminderRepo) Create(ctx context.Context, m *model.Reminder) error {
	return r.db.WithContext(ctx).Create(m).Error
}

// FindByID 按 ID 查找提醒。
func (r *ReminderRepo) FindByID(ctx context.Context, id string) (*model.Reminder, error) {
	var m model.Reminder
	if err := r.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&m).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}
	return &m, nil
}

// List 按家庭与状态查询提醒。
func (r *ReminderRepo) List(ctx context.Context, q repository.ReminderQuery) ([]*model.Reminder, error) {
	tx := r.db.WithContext(ctx).Model(&model.Reminder{}).Where("family_id = ? AND deleted_at IS NULL", q.FamilyID)
	if q.CatID != "" {
		tx = tx.Where("cat_id = ?", q.CatID)
	}
	if q.Status != "" {
		tx = tx.Where("state = ?", q.Status)
	}

	var out []*model.Reminder
	if err := tx.Order("state ASC, created_at DESC").Limit(500).Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

// ListScheduled 用稳定游标读取计划提醒，供巡检完整扫描到期窗口。
func (r *ReminderRepo) ListScheduled(ctx context.Context, q repository.ReminderScheduleQuery) ([]*model.Reminder, error) {
	if q.FamilyID == "" || q.Limit < 1 || q.Limit > 1000 {
		return nil, repository.ErrInvalidQuery
	}
	tx := r.db.WithContext(ctx).Model(&model.Reminder{}).
		Where("family_id = ? AND deleted_at IS NULL AND scheduled_at IS NOT NULL", q.FamilyID)
	if q.State != "" {
		tx = tx.Where("state = ?", q.State)
	}
	if len(q.Types) > 0 {
		tx = tx.Where("type IN ?", q.Types)
	}
	if q.From != nil {
		tx = tx.Where("scheduled_at >= ?", *q.From)
	}
	if q.ToExclusive != nil {
		tx = tx.Where("scheduled_at < ?", *q.ToExclusive)
	}
	if q.CursorAt != nil && q.CursorID != "" {
		tx = tx.Where("(scheduled_at, id) > (?, ?)", *q.CursorAt, q.CursorID)
	}
	var out []*model.Reminder
	if err := tx.Order("scheduled_at ASC, id ASC").Limit(q.Limit).Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

// Update 更新提醒。
func (r *ReminderRepo) Update(ctx context.Context, m *model.Reminder) error {
	return r.db.WithContext(ctx).Save(m).Error
}

// Delete 软删除提醒。
func (r *ReminderRepo) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Model(&model.Reminder{}).
		Where("id = ?", id).Update("deleted_at", time.Now().UTC()).Error
}
