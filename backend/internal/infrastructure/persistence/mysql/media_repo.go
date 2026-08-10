// Package mysql 实现 Media 仓储。
package mysql

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/meowhome/backend/internal/domain/model"
	"github.com/meowhome/backend/internal/domain/repository"
)

// MediaRepo 实现 repository.MediaRepo。
type MediaRepo struct {
	db *gorm.DB
}

// NewMediaRepo 创建 Media 仓储。
func NewMediaRepo(db *gorm.DB) *MediaRepo {
	return &MediaRepo{db: db}
}

var _ repository.MediaRepo = (*MediaRepo)(nil)

// Create 创建媒体资产。
func (r *MediaRepo) Create(ctx context.Context, m *model.Media) error {
	return r.db.WithContext(ctx).Create(m).Error
}

// FindByID 按 ID 查找媒体资产（含软删除）。
func (r *MediaRepo) FindByID(ctx context.Context, id string) (*model.Media, error) {
	var m model.Media
	if err := r.db.WithContext(ctx).Unscoped().Where("id = ?", id).First(&m).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

// Update 更新媒体资产。
func (r *MediaRepo) Update(ctx context.Context, m *model.Media) error {
	return r.db.WithContext(ctx).Save(m).Error
}

// Delete 软删除媒体资产。
func (r *MediaRepo) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Model(&model.Media{}).Where("id = ?", id).Update("deleted_at", time.Now().UTC()).Error
}
