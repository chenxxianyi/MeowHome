// Package mysql 实现 User 仓储。
package mysql

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/meowhome/backend/internal/domain/model"
	"github.com/meowhome/backend/internal/domain/repository"
)

// UserRepo 实现 repository.UserRepo。
type UserRepo struct {
	db *gorm.DB
}

// NewUserRepo 创建 User 仓储。
func NewUserRepo(db *gorm.DB) *UserRepo {
	return &UserRepo{db: db}
}

var _ repository.UserRepo = (*UserRepo)(nil)

// Create 创建用户。
func (r *UserRepo) Create(ctx context.Context, u *model.User) error {
	err := r.db.WithContext(ctx).Create(u).Error
	if isDuplicate(err) {
		return repository.ErrDuplicateKey
	}
	return err
}

// FindByID 按 ID 查找用户（含软删除）。
func (r *UserRepo) FindByID(ctx context.Context, id string) (*model.User, error) {
	var u model.User
	err := r.db.WithContext(ctx).Unscoped().Where("id = ?", id).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, repository.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// FindByEmail 按邮箱查找用户（含软删除）。
func (r *UserRepo) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	var u model.User
	err := r.db.WithContext(ctx).Unscoped().Where("email = ?", email).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, repository.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}
