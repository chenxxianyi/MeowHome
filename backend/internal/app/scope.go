// Package app 提供跨服务的家庭授权校验。
package app

import (
	"context"

	"github.com/meowhome/backend/internal/domain/repository"
	"github.com/meowhome/backend/internal/platform/errors"
)

// requireFamilyAccess 校验用户是否为该家庭成员，否则返回 FAMILY_FORBIDDEN。
// 所有按 family 归集的业务域都必须先过这一关，否则跨家庭数据会互相泄露。
func requireFamilyAccess(ctx context.Context, members repository.MemberRepo, familyID, userID string) error {
	if familyID == "" {
		return errors.InvalidRequest(errors.CodeValidationFailed, "family_id is required")
	}
	list, err := members.FindByUser(ctx, userID)
	if err != nil {
		return errors.NotFound(errors.CodeNotFound, "family not found")
	}
	for _, m := range list {
		if m.FamilyID == familyID {
			return nil
		}
	}
	return errors.Forbidden(errors.CodeFamilyForbidden, "access denied")
}

// memberRole 返回用户在该家庭中的角色；不是成员时返回空串。
func memberRole(ctx context.Context, members repository.MemberRepo, familyID, userID string) string {
	list, err := members.FindByUser(ctx, userID)
	if err != nil {
		return ""
	}
	for _, m := range list {
		if m.FamilyID == familyID {
			return m.Role
		}
	}
	return ""
}
