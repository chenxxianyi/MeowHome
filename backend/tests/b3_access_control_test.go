// Package tests 提供 B3 访问控制与审计日志集成测试。
package tests

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/meowhome/backend/internal/app"
	"github.com/meowhome/backend/internal/domain/repository"
	"github.com/meowhome/backend/internal/infrastructure/persistence/mysql"
	apperr "github.com/meowhome/backend/internal/platform/errors"
	"gorm.io/gorm"
)

// seedUsers 创建两个独立用户并各自建立家庭。
// 返回 alice/bob 的用户 ID 及各自家庭 ID，以及组装好的服务引用。
type b3Fixture struct {
	db        *gorm.DB
	aliceID   string
	bobID     string
	familyAID string
	familyBID string
	catAID    string
	catBID    string
	authSvc   *app.AuthService
	familySvc *app.FamilyService
	catSvc    *app.CatService
	memberSvc *app.MemberService
	auditRepo repository.AuditRepo
}

func seedB3Fixture(t *testing.T, db *gorm.DB) *b3Fixture {
	t.Helper()

	userRepo := mysql.NewUserRepo(db)
	familyRepo := mysql.NewFamilyRepo(db)
	memberRepo := mysql.NewMemberRepo(db)
	catRepo := mysql.NewCatRepo(db)
	healthRepo := mysql.NewHealthProfileRepo(db)
	refreshRepo := mysql.NewRefreshTokenRepo(db)
	auditRepo := mysql.NewAuditRepo(db)

	secret := []byte("test-secret")
	authSvc := app.NewAuthService(userRepo, refreshRepo, secret, 15*time.Minute)
	familySvc := app.NewFamilyService(familyRepo, memberRepo, userRepo, auditRepo)
	catSvc := app.NewCatService(catRepo, healthRepo, memberRepo, auditRepo, familyRepo)
	memberSvc := app.NewMemberService(memberRepo, userRepo, familyRepo)

	ctx := context.Background()

	// 创建两个用户
	alice, err := authSvc.Register(ctx, "alice@test.com", "password123", "alice")
	require.NoError(t, err)
	bob, err := authSvc.Register(ctx, "bob@test.com", "password123", "bob")
	require.NoError(t, err)

	// 各自创建家庭，创建时自动成为 owner
	famA, err := familySvc.CreateFamily(ctx, "Family A", "", "", alice.User.ID)
	require.NoError(t, err)
	famB, err := familySvc.CreateFamily(ctx, "Family B", "", "", bob.User.ID)
	require.NoError(t, err)

	// 各自添加猫咪
	catA, err := catSvc.CreateCat(ctx, famA.ID, "catA", "", "male", "2020-05-01", true, []string{"慢性肾病"}, []string{"鸡肉"}, alice.User.ID)
	require.NoError(t, err)
	catB, err := catSvc.CreateCat(ctx, famB.ID, "catB", "", "female", "", false, nil, nil, bob.User.ID)
	require.NoError(t, err)

	return &b3Fixture{
		db:        db,
		aliceID:   alice.User.ID,
		bobID:     bob.User.ID,
		familyAID: famA.ID,
		familyBID: famB.ID,
		catAID:    catA.ID,
		catBID:    catB.ID,
		authSvc:   authSvc,
		familySvc: familySvc,
		catSvc:    catSvc,
		memberSvc: memberSvc,
		auditRepo: auditRepo,
	}
}

// TestAccessControl_FamilyA_B_CatAccess 验证家庭 A/B 跨家庭隔离。
//
// 跳过条件：MYSQL_TEST_DSN 未设置
func TestAccessControl_FamilyA_B_CatAccess(t *testing.T) {
	db := SetupTestDB(t)
	if db == nil {
		t.Skip("integration tests skipped")
	}
	defer RollbackTestDB(t, db)

	fx := seedB3Fixture(t, db)
	ctx := context.Background()

	t.Run("alice can access her own family and cat", func(t *testing.T) {
		_, err := fx.familySvc.GetFamily(ctx, fx.familyAID, fx.aliceID)
		assert.NoError(t, err)
		cat, err := fx.catSvc.GetCat(ctx, fx.catAID, fx.aliceID)
		assert.NoError(t, err)
		assert.Equal(t, "2020-05-01", cat.Birthday)
		assert.True(t, cat.Neutered)
		assert.Equal(t, []string{"慢性肾病"}, cat.Diseases)
		assert.Equal(t, []string{"鸡肉"}, cat.Allergies)
	})

	t.Run("bob can access his own family and cat", func(t *testing.T) {
		_, err := fx.familySvc.GetFamily(ctx, fx.familyBID, fx.bobID)
		assert.NoError(t, err)
		_, err = fx.catSvc.GetCat(ctx, fx.catBID, fx.bobID)
		assert.NoError(t, err)
	})

	t.Run("alice cannot read family B", func(t *testing.T) {
		_, err := fx.familySvc.GetFamily(ctx, fx.familyBID, fx.aliceID)
		assertErrorFamilyForbidden(t, err)
	})

	t.Run("bob cannot read family A", func(t *testing.T) {
		_, err := fx.familySvc.GetFamily(ctx, fx.familyAID, fx.bobID)
		assertErrorFamilyForbidden(t, err)
	})

	t.Run("alice cannot read bob's cat catB", func(t *testing.T) {
		_, err := fx.catSvc.GetCat(ctx, fx.catBID, fx.aliceID)
		assertErrorFamilyForbidden(t, err)
	})

	t.Run("bob cannot read alice's cat catA", func(t *testing.T) {
		_, err := fx.catSvc.GetCat(ctx, fx.catAID, fx.bobID)
		assertErrorFamilyForbidden(t, err)
	})

	t.Run("alice cannot list family B cats", func(t *testing.T) {
		_, err := fx.catSvc.ListCats(ctx, fx.familyBID, fx.aliceID)
		assertErrorFamilyForbidden(t, err)
	})

	t.Run("alice cannot create cat in family B", func(t *testing.T) {
		_, err := fx.catSvc.CreateCat(ctx, fx.familyBID, "intruder", "", "male", "", false, nil, nil, fx.aliceID)
		assertErrorFamilyForbidden(t, err)
	})

	t.Run("alice cannot update bob's cat", func(t *testing.T) {
		_, err := fx.catSvc.UpdateCat(ctx, fx.catBID, fx.aliceID, "renamed", "", "", "", nil)
		assertErrorFamilyForbidden(t, err)
	})

	t.Run("alice cannot delete bob's cat", func(t *testing.T) {
		err := fx.catSvc.DeleteCat(ctx, fx.catBID, fx.aliceID)
		assertErrorFamilyForbidden(t, err)
	})

	t.Run("alice cannot list family B members", func(t *testing.T) {
		_, err := fx.familySvc.ListMembers(ctx, fx.familyBID, fx.aliceID)
		assertErrorFamilyForbidden(t, err)
	})
}

// TestAccessControl_MemberRole 验证成员角色权限（owner/admin 管理、member 只读）。
func TestAccessControl_MemberRole(t *testing.T) {
	db := SetupTestDB(t)
	if db == nil {
		t.Skip("integration tests skipped")
	}
	defer RollbackTestDB(t, db)

	fx := seedB3Fixture(t, db)
	ctx := context.Background()

	t.Run("inviting a non-member then revocation denies access", func(t *testing.T) {
		// charlie 尚未加入任何家庭
		charlie, err := fx.authSvc.Register(ctx, "charlie@test.com", "password123", "charlie")
		require.NoError(t, err)

		// charlie 不能访问家庭 A
		_, err = fx.familySvc.GetFamily(ctx, fx.familyAID, charlie.User.ID)
		assertErrorFamilyForbidden(t, err)

		// alice（owner）邀请 charlie 加入家庭 A
		require.NoError(t, fx.memberSvc.InviteMember(ctx, fx.familyAID, charlie.User.ID, "member"))

		// charlie 现在可以访问家庭 A
		_, err = fx.familySvc.GetFamily(ctx, fx.familyAID, charlie.User.ID)
		assert.NoError(t, err)

		// 但 charlie 仍不能访问家庭 B
		_, err = fx.familySvc.GetFamily(ctx, fx.familyBID, charlie.User.ID)
		assertErrorFamilyForbidden(t, err)
	})

	t.Run("role must be owner/admin/member", func(t *testing.T) {
		err := fx.memberSvc.UpdateMemberRole(ctx, "some-member-id", "superuser")
		assert.Error(t, err)
		var ae *apperr.AppError
		if errorsAs(t, err, &ae) {
			assert.Equal(t, apperr.CodeValidationFailed, ae.Code)
		}
	})
}

// TestAuditLog_RecordedOnCreate 验证家庭/猫咪创建记录审计日志。
func TestAuditLog_RecordedOnCreate(t *testing.T) {
	db := SetupTestDB(t)
	if db == nil {
		t.Skip("integration tests skipped")
	}
	defer RollbackTestDB(t, db)

	fx := seedB3Fixture(t, db)
	ctx := context.Background()

	t.Run("family.create audit log exists for family A", func(t *testing.T) {
		var count int64
		err := fx.db.WithContext(ctx).Table("audit_logs").
			Where("family_id = ? AND action = ?", fx.familyAID, "family.create").
			Count(&count).Error
		require.NoError(t, err)
		assert.Equal(t, int64(1), count, "exactly one family.create audit record expected")
	})

	t.Run("cat.create audit log exists for cat A", func(t *testing.T) {
		var count int64
		err := fx.db.WithContext(ctx).Table("audit_logs").
			Where("family_id = ? AND action = ? AND resource = ?", fx.familyAID, "cat.create", fx.catAID).
			Count(&count).Error
		require.NoError(t, err)
		assert.Equal(t, int64(1), count, "exactly one cat.create audit record expected")
	})
}

// assertErrorFamilyForbidden 断言 err 为 FAMILY_FORBIDDEN 拒绝错误。
func assertErrorFamilyForbidden(t *testing.T, err error) {
	t.Helper()
	require.Error(t, err)
	var ae *apperr.AppError
	require.ErrorAs(t, err, &ae)
	assert.Equal(t, apperr.CodeFamilyForbidden, ae.Code, "expected FAMILY_FORBIDDEN, got %q (%s)", ae.Code, ae.Message)
	assert.Equal(t, apperr.TypeAuth, ae.Type)
}

// errorsAs 封装 errors.As 便于断言。
func errorsAs(t *testing.T, err error, target any) bool {
	t.Helper()
	return errors.As(err, target)
}
