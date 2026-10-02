package app

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/meowhome/backend/internal/domain/model"
	"github.com/meowhome/backend/internal/domain/repository"
	"github.com/meowhome/backend/internal/platform/errors"
)

// ReminderService 提醒服务。
type ReminderService struct {
	reminders repository.ReminderRepo
	members   repository.MemberRepo
	cats      repository.CatRepo
	families  repository.FamilyRepo
}

// NewReminderService 创建提醒服务。
func NewReminderService(reminders repository.ReminderRepo, members repository.MemberRepo, cats ...repository.CatRepo) *ReminderService {
	var catRepo repository.CatRepo
	if len(cats) > 0 {
		catRepo = cats[0]
	}
	return &ReminderService{reminders: reminders, members: members, cats: catRepo}
}

// NewReminderServiceWithRepos 注入猫咪与家庭仓储，供需要校验归属和展示时区的完整路径使用。
func NewReminderServiceWithRepos(reminders repository.ReminderRepo, members repository.MemberRepo, cats repository.CatRepo, families repository.FamilyRepo) *ReminderService {
	return &ReminderService{reminders: reminders, members: members, cats: cats, families: families}
}

// ReminderEnvelope 提醒响应（字段名与前端 Reminder 类型对齐）。
type ReminderEnvelope struct {
	ID          string     `json:"id"`
	CatID       string     `json:"catId"`
	Type        string     `json:"type"`
	Title       string     `json:"title"`
	Subtitle    string     `json:"subtitle"`
	Time        string     `json:"time"`
	State       string     `json:"state"`
	Icon        string     `json:"icon"`
	Rule        string     `json:"rule,omitempty"`
	ScheduledAt *time.Time `json:"scheduled_at,omitempty"`
	Timezone    string     `json:"timezone,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

// ReminderInput 创建提醒的输入。
type ReminderInput struct {
	CatID       string     `json:"catId"`
	Type        string     `json:"type"`
	Title       string     `json:"title"`
	Subtitle    string     `json:"subtitle"`
	Time        string     `json:"time"`
	Icon        string     `json:"icon"`
	Rule        string     `json:"rule"`
	ScheduledAt *time.Time `json:"scheduled_at,omitempty"`
	Timezone    string     `json:"timezone,omitempty"`
}

// List 列出提醒；state 为空表示不限（todo / done）。
func (s *ReminderService) List(ctx context.Context, familyID, userID, state string) ([]*ReminderEnvelope, error) {
	if err := requireFamilyAccess(ctx, s.members, familyID, userID); err != nil {
		return nil, err
	}
	rows, err := s.reminders.List(ctx, repository.ReminderQuery{FamilyID: familyID, Status: state})
	if err != nil {
		return nil, errors.Wrap(errors.TypeInternal, errors.CodeInvalidRequest, "failed to list reminders", err)
	}
	out := make([]*ReminderEnvelope, 0, len(rows))
	for _, r := range rows {
		out = append(out, toReminderEnvelope(r))
	}
	return out, nil
}

// Complete 标记提醒为已完成。
func (s *ReminderService) Complete(ctx context.Context, familyID, userID, id string) (*ReminderEnvelope, error) {
	if err := requireFamilyAccess(ctx, s.members, familyID, userID); err != nil {
		return nil, err
	}
	rem, err := s.reminders.FindByID(ctx, id)
	if err != nil {
		return nil, errors.NotFound(errors.CodeNotFound, "reminder not found")
	}
	if rem.FamilyID != familyID {
		return nil, errors.Forbidden(errors.CodeFamilyForbidden, "access denied")
	}

	rem.State = "done"
	now := time.Now().UTC()
	rem.UpdatedAt = now
	rem.CompletedAt = &now
	if err := s.reminders.Update(ctx, rem); err != nil {
		return nil, errors.Wrap(errors.TypeInternal, errors.CodeInvalidRequest, "failed to update reminder", err)
	}
	return toReminderEnvelope(rem), nil
}

// Create 创建提醒。
func (s *ReminderService) Create(ctx context.Context, familyID, userID string, in ReminderInput) (*ReminderEnvelope, error) {
	if err := requireFamilyAccess(ctx, s.members, familyID, userID); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	var err error
	in, err = normalizeReminderInput(ctx, s.cats, s.families, familyID, in, now, false, 255)
	if err != nil {
		return nil, err
	}
	rem := &model.Reminder{
		Base:        model.Base{ID: model.NewBase(familyID, userID).ID, CreatedBy: userID, CreatedAt: now, UpdatedAt: now},
		FamilyID:    familyID,
		CatID:       in.CatID,
		Type:        in.Type,
		Title:       in.Title,
		Subtitle:    in.Subtitle,
		TimeLabel:   in.Time,
		State:       "todo",
		Icon:        in.Icon,
		Rule:        in.Rule,
		ScheduledAt: in.ScheduledAt,
		Timezone:    in.Timezone,
	}
	if err := s.reminders.Create(ctx, rem); err != nil {
		return nil, errors.Wrap(errors.TypeInternal, errors.CodeInvalidRequest, "failed to create reminder", err)
	}
	return toReminderEnvelope(rem), nil
}

// normalizeReminderInput 是普通提醒和 Agent 草稿确认共用的业务校验。
// 旧提醒 API 保留未填猫咪时默认为 both；Agent 必须显式指定归属。
func normalizeReminderInput(ctx context.Context, cats repository.CatRepo, families repository.FamilyRepo, familyID string, in ReminderInput, now time.Time, requireExplicitCat bool, maxTitle int) (ReminderInput, error) {
	in.Title = strings.TrimSpace(in.Title)
	if in.Title == "" || utf8.RuneCountInString(in.Title) > maxTitle {
		return in, errors.InvalidRequest(errors.CodeValidationFailed, "reminder title length is invalid")
	}
	if in.Type == "" {
		in.Type = "custom"
	}
	if in.CatID == "" {
		if requireExplicitCat {
			return in, errors.InvalidRequest(errors.CodeValidationFailed, "cat_id is required; use both for a family reminder")
		}
		in.CatID = "both"
	}
	if in.CatID != "both" {
		if cats == nil {
			return in, errors.New(errors.TypeInternal, errors.CodeInvalidRequest, "cat repository is unavailable")
		}
		cat, err := cats.FindByID(ctx, in.CatID)
		if err != nil || cat.FamilyID != familyID || cat.DeletedAt != nil {
			return in, errors.Forbidden(errors.CodeFamilyForbidden, "cat does not belong to family")
		}
	}
	if in.ScheduledAt != nil {
		if !in.ScheduledAt.After(now) {
			return in, errors.InvalidRequest(errors.CodeValidationFailed, "scheduled_at must be in the future")
		}
		utc := in.ScheduledAt.UTC()
		in.ScheduledAt = &utc
		if in.Timezone == "" && families != nil {
			if family, err := families.FindByID(ctx, familyID); err == nil && family.DeletedAt == nil {
				in.Timezone = family.Timezone
			}
		}
		if in.Timezone == "" {
			in.Timezone = "UTC"
		}
	}
	if in.Timezone != "" {
		if _, err := time.LoadLocation(in.Timezone); err != nil {
			return in, errors.InvalidRequest(errors.CodeValidationFailed, "invalid timezone")
		}
	}
	return in, nil
}

func toReminderEnvelope(r *model.Reminder) *ReminderEnvelope {
	return &ReminderEnvelope{
		ID:          r.ID,
		CatID:       r.CatID,
		Type:        r.Type,
		Title:       r.Title,
		Subtitle:    r.Subtitle,
		Time:        r.TimeLabel,
		State:       r.State,
		Icon:        r.Icon,
		Rule:        r.Rule,
		ScheduledAt: r.ScheduledAt,
		Timezone:    r.Timezone,
		CompletedAt: r.CompletedAt,
	}
}
