// Package app 提供应用层成员服务。
package app

import (
	"context"
	"time"

	"github.com/meowhome/backend/internal/domain/model"
	"github.com/meowhome/backend/internal/domain/repository"
	"github.com/meowhome/backend/internal/platform/errors"
)

// MemberService 成员服务（邀请/角色/移除）。
type MemberService struct {
	memberRepo repository.MemberRepo
	userRepo   repository.UserRepo
	familyRepo repository.FamilyRepo
}

// NewMemberService 创建成员服务。
func NewMemberService(memberRepo repository.MemberRepo, userRepo repository.UserRepo, familyRepo repository.FamilyRepo) *MemberService {
	return &MemberService{
		memberRepo: memberRepo,
		userRepo:   userRepo,
		familyRepo: familyRepo,
	}
}

// InviteMember 邀请成员（注册新用户并关联到家庭）。
func (s *MemberService) InviteMember(ctx context.Context, familyID, userID, role string) error {
	if role == "" {
		role = "member"
	}
	_, err := s.familyRepo.FindByID(ctx, familyID)
	if err != nil {
		return errors.NotFound(errors.CodeNotFound, "family not found")
	}
	_, err = s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return errors.NotFound(errors.CodeNotFound, "user not found")
	}
	now := time.Now().UTC()
	m := &model.Member{
		Base: model.Base{
			ID:        model.NewBase(familyID, "").ID,
			CreatedBy: userID,
			CreatedAt: now,
			UpdatedAt: now,
		},
		FamilyID: familyID,
		UserID:   userID,
		Role:     role,
	}
	if err := s.memberRepo.Create(ctx, m); err != nil {
		if isDuplicate(err) {
			return errors.Conflict(errors.CodeConflict, "user already a member")
		}
		return errors.Wrap(errors.TypeInternal, errors.CodeInvalidRequest, "failed to invite member", err)
	}
	return nil
}

// UpdateMemberRole 更新角色。
func (s *MemberService) UpdateMemberRole(ctx context.Context, memberID, role string) error {
	if role != "owner" && role != "admin" && role != "member" {
		return errors.InvalidRequest(errors.CodeValidationFailed, "role must be owner/admin/member")
	}
	if err := s.memberRepo.UpdateRole(ctx, memberID, role); err != nil {
		return errors.Wrap(errors.TypeInternal, errors.CodeInvalidRequest, "failed to update role", err)
	}
	return nil
}

// RemoveMember 移除成员。
func (s *MemberService) RemoveMember(ctx context.Context, memberID string) error {
	if err := s.memberRepo.Delete(ctx, memberID); err != nil {
		return errors.Wrap(errors.TypeInternal, errors.CodeInvalidRequest, "failed to remove member", err)
	}
	return nil
}
