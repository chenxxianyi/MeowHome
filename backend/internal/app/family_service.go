// Package app 提供应用层家庭与猫咪服务。
package app

import (
	"context"
	"encoding/json"
	"time"

	"github.com/meowhome/backend/internal/domain/model"
	"github.com/meowhome/backend/internal/domain/repository"
	"github.com/meowhome/backend/internal/platform/errors"
)

// FamilyService 家庭服务。
type FamilyService struct {
	familyRepo repository.FamilyRepo
	memberRepo repository.MemberRepo
	userRepo   repository.UserRepo
	auditRepo  repository.AuditRepo
}

// NewFamilyService 创建家庭服务。
func NewFamilyService(
	familyRepo repository.FamilyRepo,
	memberRepo repository.MemberRepo,
	userRepo repository.UserRepo,
	auditRepo repository.AuditRepo,
) *FamilyService {
	return &FamilyService{
		familyRepo: familyRepo,
		memberRepo: memberRepo,
		userRepo:   userRepo,
		auditRepo:  auditRepo,
	}
}

// FamilyEnvelope 家庭响应 DTO。
type FamilyEnvelope struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Timezone  string  `json:"timezone"`
	Currency  string  `json:"currency"`
	AvatarURL *string `json:"avatar_url,omitempty"`
}

// MemberEnvelope 成员响应 DTO。
type MemberEnvelope struct {
	ID       string `json:"id"`
	FamilyID string `json:"family_id"`
	UserID   string `json:"user_id"`
	UserName string `json:"user_name"`
	Role     string `json:"role"`
}

// CreateFamily 创建家庭（并自动建立 owner 成员关系）。
func (s *FamilyService) CreateFamily(ctx context.Context, name, timezone, currency string, creatorUserID string) (*model.Family, error) {
	if name == "" {
		return nil, errors.InvalidRequest(errors.CodeValidationFailed, "name is required")
	}
	if timezone == "" {
		timezone = "Asia/Shanghai"
	}
	if currency == "" {
		currency = "CNY"
	}
	now := time.Now().UTC()
	f := &model.Family{
		Base: model.Base{
			ID:        model.NewBase("", "").ID,
			CreatedBy: creatorUserID,
			CreatedAt: now,
			UpdatedAt: now,
		},
		Name:     name,
		Timezone: timezone,
		Currency: currency,
	}
	if err := s.familyRepo.Create(ctx, f); err != nil {
		if isDuplicate(err) {
			return nil, errors.Conflict(errors.CodeConflict, "family name already exists")
		}
		return nil, errors.Wrap(errors.TypeInternal, errors.CodeInvalidRequest, "failed to create family", err)
	}

	// 自动建立 owner 成员关系
	m := &model.Member{
		Base: model.Base{
			ID:        model.NewBase("", "").ID,
			CreatedBy: creatorUserID,
			CreatedAt: now,
			UpdatedAt: now,
		},
		FamilyID: f.ID,
		UserID:   creatorUserID,
		Role:     "owner",
	}
	if err := s.memberRepo.Create(ctx, m); err != nil {
		return nil, errors.Wrap(errors.TypeInternal, errors.CodeInvalidRequest, "failed to create member", err)
	}

	// 审计日志
	_ = s.auditRepo.Create(ctx, &model.AuditLog{
		Base: model.Base{
			ID:        model.NewBase("", "").ID,
			CreatedBy: creatorUserID,
			CreatedAt: now,
			UpdatedAt: now,
		},
		FamilyID: f.ID,
		UserID:   creatorUserID,
		Action:   "family.create",
		Resource: f.ID,
		Detail:   "family created",
	})

	return f, nil
}

// enforceAccess 校验用户是否属于目标家庭（B3 简化版：所有成员均可访问）。
func (s *FamilyService) enforceAccess(ctx context.Context, familyID, userID string) error {
	members, err := s.memberRepo.FindByUser(ctx, userID)
	if err != nil {
		return errors.NotFound(errors.CodeNotFound, "family not found")
	}
	for _, m := range members {
		if m.FamilyID == familyID {
			return nil
		}
	}
	return errors.Forbidden(errors.CodeFamilyForbidden, "access denied")
}

// GetFamily 获取家庭详情。
func (s *FamilyService) GetFamily(ctx context.Context, familyID, userID string) (*model.Family, error) {
	if err := s.enforceAccess(ctx, familyID, userID); err != nil {
		return nil, err
	}
	f, err := s.familyRepo.FindByID(ctx, familyID)
	if err != nil {
		return nil, errors.NotFound(errors.CodeNotFound, "family not found")
	}
	return f, nil
}

// UpdateFamily 更新家庭（仅 owner/admin）。
func (s *FamilyService) UpdateFamily(ctx context.Context, familyID, userID string, name, timezone, currency string) (*model.Family, error) {
	if err := s.enforceAccess(ctx, familyID, userID); err != nil {
		return nil, err
	}
	f, err := s.familyRepo.FindByID(ctx, familyID)
	if err != nil {
		return nil, errors.NotFound(errors.CodeNotFound, "family not found")
	}
	if name != "" {
		f.Name = name
	}
	if timezone != "" {
		f.Timezone = timezone
	}
	if currency != "" {
		f.Currency = currency
	}
	f.UpdatedAt = time.Now().UTC()
	if err := s.familyRepo.Update(ctx, f); err != nil {
		return nil, errors.Wrap(errors.TypeInternal, errors.CodeInvalidRequest, "failed to update family", err)
	}
	return f, nil
}

// ListMembers 列出家庭所有成员。
func (s *FamilyService) ListMembers(ctx context.Context, familyID, userID string) ([]*MemberEnvelope, error) {
	if err := s.enforceAccess(ctx, familyID, userID); err != nil {
		return nil, err
	}
	members, err := s.memberRepo.FindByFamily(ctx, familyID)
	if err != nil {
		return nil, errors.Wrap(errors.TypeInternal, errors.CodeInvalidRequest, "failed to load members", err)
	}
	result := make([]*MemberEnvelope, 0, len(members))
	for _, member := range members {
		name := member.UserID
		if user, findErr := s.userRepo.FindByID(ctx, member.UserID); findErr == nil && user != nil && user.Name != "" {
			name = user.Name
		}
		result = append(result, &MemberEnvelope{
			ID:       member.ID,
			FamilyID: member.FamilyID,
			UserID:   member.UserID,
			UserName: name,
			Role:     member.Role,
		})
	}
	return result, nil
}

// CatService 猫咪服务。
type CatService struct {
	catRepo    repository.CatRepo
	healthRepo repository.CatHealthProfileRepo
	memberRepo repository.MemberRepo
	auditRepo  repository.AuditRepo
	familyRepo repository.FamilyRepo
}

// NewCatService 创建猫咪服务。
func NewCatService(catRepo repository.CatRepo, healthRepo repository.CatHealthProfileRepo, memberRepo repository.MemberRepo, auditRepo repository.AuditRepo, familyRepo repository.FamilyRepo) *CatService {
	return &CatService{
		catRepo:    catRepo,
		healthRepo: healthRepo,
		memberRepo: memberRepo,
		auditRepo:  auditRepo,
		familyRepo: familyRepo,
	}
}

// CatEnvelope 猫咪响应 DTO。
type CatEnvelope struct {
	ID        string  `json:"id"`
	FamilyID  string  `json:"family_id"`
	Name      string  `json:"name"`
	Breed     string  `json:"breed"`
	Gender    string  `json:"gender"`
	BirthDate *string `json:"birth_date,omitempty"`
	Neutered  bool    `json:"neutered"`
	AvatarKey *string `json:"avatar_key,omitempty"`
}

// enforceCatAccess 校验用户是否属于目标猫咪所属家庭（B3 简化版：所有成员均可访问）。
func (s *CatService) enforceCatAccess(ctx context.Context, familyID, userID string) error {
	members, err := s.memberRepo.FindByUser(ctx, userID)
	if err != nil {
		return errors.NotFound(errors.CodeNotFound, "family not found")
	}
	for _, m := range members {
		if m.FamilyID == familyID {
			return nil
		}
	}
	return errors.Forbidden(errors.CodeFamilyForbidden, "access denied")
}

// CreateCat 创建猫咪。
func (s *CatService) CreateCat(ctx context.Context, familyID, name, breed, gender, birthday string, neutered bool, diseases, allergies []string, creatorUserID string) (*model.Cat, error) {
	if err := s.enforceCatAccess(ctx, familyID, creatorUserID); err != nil {
		return nil, err
	}
	if _, err := s.familyRepo.FindByID(ctx, familyID); err != nil {
		return nil, errors.NotFound(errors.CodeNotFound, "family not found")
	}
	if name == "" {
		return nil, errors.InvalidRequest(errors.CodeValidationFailed, "name is required")
	}
	if gender == "" {
		gender = "unknown"
	}
	now := time.Now().UTC()
	c := &model.Cat{
		Base: model.Base{
			ID:        model.NewBase(familyID, "").ID,
			CreatedBy: creatorUserID,
			CreatedAt: now,
			UpdatedAt: now,
		},
		FamilyID:  familyID,
		Name:      name,
		Breed:     breed,
		Gender:    gender,
		Birthday:  birthday,
		Neutered:  neutered,
		Diseases:  diseases,
		Allergies: allergies,
	}
	if err := s.catRepo.Create(ctx, c); err != nil {
		return nil, errors.Wrap(errors.TypeInternal, errors.CodeInvalidRequest, "failed to create cat", err)
	}
	diseasesJSON, _ := json.Marshal(diseases)
	allergiesJSON, _ := json.Marshal(allergies)
	if err := s.healthRepo.Create(ctx, &model.CatHealthProfile{
		Base:              model.NewBase("", creatorUserID),
		CatID:             c.ID,
		Diseases:          string(diseasesJSON),
		Allergies:         string(allergiesJSON),
		Contraindications: "[]",
	}); err != nil {
		_ = s.catRepo.Delete(ctx, c.ID)
		return nil, errors.Wrap(errors.TypeInternal, errors.CodeInvalidRequest, "failed to create cat health profile", err)
	}

	// 审计日志
	_ = s.auditRepo.Create(ctx, &model.AuditLog{
		Base: model.Base{
			ID:        model.NewBase(familyID, "").ID,
			CreatedBy: creatorUserID,
			CreatedAt: now,
			UpdatedAt: now,
		},
		FamilyID: familyID,
		UserID:   creatorUserID,
		Action:   "cat.create",
		Resource: c.ID,
		Detail:   "cat created",
	})

	return c, nil
}

// GetCat 获取猫咪详情。
func (s *CatService) GetCat(ctx context.Context, catID, userID string) (*model.Cat, error) {
	c, err := s.catRepo.FindByID(ctx, catID)
	if err != nil {
		return nil, errors.NotFound(errors.CodeNotFound, "cat not found")
	}
	if err := s.enforceCatAccess(ctx, c.FamilyID, userID); err != nil {
		return nil, err
	}
	if err := s.loadCatHealth(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

// UpdateCat 更新猫咪。
func (s *CatService) UpdateCat(ctx context.Context, catID, userID string, name, breed, gender, birthday string, neutered *bool) (*model.Cat, error) {
	c, err := s.catRepo.FindByID(ctx, catID)
	if err != nil {
		return nil, errors.NotFound(errors.CodeNotFound, "cat not found")
	}
	if err := s.enforceCatAccess(ctx, c.FamilyID, userID); err != nil {
		return nil, err
	}
	if name != "" {
		c.Name = name
	}
	if breed != "" {
		c.Breed = breed
	}
	if gender != "" {
		c.Gender = gender
	}
	if birthday != "" {
		c.Birthday = birthday
	}
	if neutered != nil {
		c.Neutered = *neutered
	}
	c.UpdatedAt = time.Now().UTC()
	if err := s.catRepo.Update(ctx, c); err != nil {
		return nil, errors.Wrap(errors.TypeInternal, errors.CodeInvalidRequest, "failed to update cat", err)
	}
	if err := s.loadCatHealth(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

// ListCats 列出家庭所有猫咪。
func (s *CatService) ListCats(ctx context.Context, familyID, userID string) ([]*model.Cat, error) {
	if err := s.enforceCatAccess(ctx, familyID, userID); err != nil {
		return nil, err
	}
	cats, err := s.catRepo.ListByFamily(ctx, familyID)
	if err != nil {
		return nil, err
	}
	for _, cat := range cats {
		if err := s.loadCatHealth(ctx, cat); err != nil {
			return nil, err
		}
	}
	return cats, nil
}

func (s *CatService) loadCatHealth(ctx context.Context, cat *model.Cat) error {
	profile, err := s.healthRepo.FindByCatID(ctx, cat.ID)
	if err != nil {
		return errors.Wrap(errors.TypeInternal, errors.CodeInvalidRequest, "failed to load cat health profile", err)
	}
	if profile == nil {
		cat.Diseases = []string{}
		cat.Allergies = []string{}
		return nil
	}
	cat.Diseases = decodeStringList(profile.Diseases)
	cat.Allergies = decodeStringList(profile.Allergies)
	return nil
}

func decodeStringList(raw string) []string {
	items := []string{}
	if raw != "" {
		_ = json.Unmarshal([]byte(raw), &items)
	}
	return items
}

// DeleteCat 软删除猫咪。
func (s *CatService) DeleteCat(ctx context.Context, catID, userID string) error {
	c, err := s.catRepo.FindByID(ctx, catID)
	if err != nil {
		return errors.NotFound(errors.CodeNotFound, "cat not found")
	}
	if err := s.enforceCatAccess(ctx, c.FamilyID, userID); err != nil {
		return err
	}
	return s.catRepo.Delete(ctx, catID)
}
