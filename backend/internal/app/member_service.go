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

// MeResult 当前用户信息（对应 openapi 的 User schema）。
type MeResult struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	UserName string `json:"user_name"`
	FamilyID string `json:"family_id,omitempty"`
	MemberID string `json:"member_id,omitempty"`
	Role     string `json:"role,omitempty"`
	Timezone string `json:"timezone,omitempty"`
}

// MyFamily 用户所属家庭摘要（含其在该家庭中的成员身份）。
type MyFamily struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Timezone string `json:"timezone"`
	Currency string `json:"currency"`
	MemberID string `json:"member_id"`
	Role     string `json:"role"`
}

// Me 返回当前用户的完整信息，并附带其首个家庭的归属与角色。
// 前端登录后需要 family_id 才能访问家庭资源，因此这里必须一并返回。
func (s *MemberService) Me(ctx context.Context, userID string) (*MeResult, error) {
	u, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, errors.NotFound(errors.CodeNotFound, "user not found")
	}

	res := &MeResult{ID: u.ID, Email: u.Email, UserName: u.Name}

	members, err := s.memberRepo.FindByUser(ctx, userID)
	if err != nil || len(members) == 0 {
		return res, nil // 尚未加入任何家庭，返回不带归属的用户信息
	}
	m := members[0]
	res.FamilyID = m.FamilyID
	res.MemberID = m.ID
	res.Role = m.Role
	res.Timezone = m.Timezone
	return res, nil
}

// ListMyFamilies 列出当前用户所属的全部家庭。
func (s *MemberService) ListMyFamilies(ctx context.Context, userID string) ([]*MyFamily, error) {
	members, err := s.memberRepo.FindByUser(ctx, userID)
	if err != nil {
		return nil, errors.Wrap(errors.TypeInternal, errors.CodeInvalidRequest, "failed to load memberships", err)
	}

	out := make([]*MyFamily, 0, len(members))
	for _, m := range members {
		f, err := s.familyRepo.FindByID(ctx, m.FamilyID)
		if err != nil {
			continue // 家庭已被删除时跳过该成员关系
		}
		out = append(out, &MyFamily{
			ID:       f.ID,
			Name:     f.Name,
			Timezone: f.Timezone,
			Currency: f.Currency,
			MemberID: m.ID,
			Role:     m.Role,
		})
	}
	return out, nil
}
