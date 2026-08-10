// Package mysql 实现 Member 仓储。
package mysql

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/meowhome/backend/internal/domain/model"
	"github.com/meowhome/backend/internal/domain/repository"
)

// MemberRepo 实现 repository.MemberRepo。
type MemberRepo struct {
	db *gorm.DB
}

// NewMemberRepo 创建 Member 仓储。
func NewMemberRepo(db *gorm.DB) *MemberRepo {
	return &MemberRepo{db: db}
}

var _ repository.MemberRepo = (*MemberRepo)(nil)

// Create 创建成员。
func (r *MemberRepo) Create(ctx context.Context, m *model.Member) error {
	return r.db.WithContext(ctx).Create(m).Error
}

// FindByFamily 查找家庭的所有成员（含软删除）。
func (r *MemberRepo) FindByFamily(ctx context.Context, familyID string) ([]*model.Member, error) {
	var members []*model.Member
	if err := r.db.WithContext(ctx).Unscoped().Where("family_id = ?", familyID).Find(&members).Error; err != nil {
		return nil, err
	}
	return members, nil
}

// FindByUser 查找用户的所有成员记录（含软删除）。
func (r *MemberRepo) FindByUser(ctx context.Context, userID string) ([]*model.Member, error) {
	var members []*model.Member
	if err := r.db.WithContext(ctx).Unscoped().Where("user_id = ?", userID).Find(&members).Error; err != nil {
		return nil, err
	}
	return members, nil
}

// UpdateRole 更新成员角色。
func (r *MemberRepo) UpdateRole(ctx context.Context, memberID, role string) error {
	return r.db.WithContext(ctx).Model(&model.Member{}).Where("id = ?", memberID).Update("role", role).Error
}

// Delete 软删除成员。
func (r *MemberRepo) Delete(ctx context.Context, memberID string) error {
	return r.db.WithContext(ctx).Model(&model.Member{}).Where("id = ?", memberID).Update("deleted_at", time.Now().UTC()).Error
}
