// Package mysql 实现 Family 仓储。
package mysql

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/meowhome/backend/internal/domain/model"
	"github.com/meowhome/backend/internal/domain/repository"
)

// FamilyRepo 实现 repository.FamilyRepo。
type FamilyRepo struct {
	db *gorm.DB
}

// NewFamilyRepo 创建 Family 仓储。
func NewFamilyRepo(db *gorm.DB) *FamilyRepo {
	return &FamilyRepo{db: db}
}

var _ repository.FamilyRepo = (*FamilyRepo)(nil)

// Create 创建家庭。
func (r *FamilyRepo) Create(ctx context.Context, f *model.Family) error {
	err := r.db.WithContext(ctx).Create(f).Error
	if isDuplicate(err) {
		return repository.ErrDuplicateKey
	}
	return err
}

// FindByID 按 ID 查找家庭（含软删除）。
func (r *FamilyRepo) FindByID(ctx context.Context, id string) (*model.Family, error) {
	var f model.Family
	err := r.db.WithContext(ctx).Unscoped().Where("id = ?", id).First(&f).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, repository.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &f, nil
}

// Update 更新家庭。
func (r *FamilyRepo) Update(ctx context.Context, f *model.Family) error {
	err := r.db.WithContext(ctx).Save(f).Error
	if isDuplicate(err) {
		return repository.ErrDuplicateKey
	}
	return err
}

// Delete 软删除家庭。
func (r *FamilyRepo) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Model(&model.Family{}).Where("id = ?", id).Update("deleted_at", time.Now().UTC()).Error
}
