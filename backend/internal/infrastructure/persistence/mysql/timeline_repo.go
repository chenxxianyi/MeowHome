// Package mysql 实现 Timeline 仓储。
package mysql

import (
	"context"

	"gorm.io/gorm"

	"github.com/meowhome/backend/internal/domain/model"
	"github.com/meowhome/backend/internal/domain/repository"
)

// TimelineRepo 实现 repository.TimelineRepo。
type TimelineRepo struct {
	db *gorm.DB
}

// NewTimelineRepo 创建 Timeline 仓储。
func NewTimelineRepo(db *gorm.DB) *TimelineRepo {
	return &TimelineRepo{db: db}
}

var _ repository.TimelineRepo = (*TimelineRepo)(nil)

// Create 创建时光事件。
func (r *TimelineRepo) Create(ctx context.Context, e *model.TimelineEvent) error {
	return r.db.WithContext(ctx).Create(e).Error
}

// ListByFamily 按家庭列出时光事件（按发生日期倒序）。
func (r *TimelineRepo) ListByFamily(ctx context.Context, familyID string, limit int) ([]*model.TimelineEvent, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	var out []*model.TimelineEvent
	if err := r.db.WithContext(ctx).Where("family_id = ?", familyID).
		Order("occurred_on DESC, created_at DESC").Limit(limit).Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}
