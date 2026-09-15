package app

import (
	"context"
	"time"

	"github.com/meowhome/backend/internal/domain/model"
	"github.com/meowhome/backend/internal/domain/repository"
	"github.com/meowhome/backend/internal/platform/errors"
)

// ReminderService 提醒服务。
type ReminderService struct {
	reminders repository.ReminderRepo
	members   repository.MemberRepo
}

// NewReminderService 创建提醒服务。
func NewReminderService(reminders repository.ReminderRepo, members repository.MemberRepo) *ReminderService {
	return &ReminderService{reminders: reminders, members: members}
}

// ReminderEnvelope 提醒响应（字段名与前端 Reminder 类型对齐）。
type ReminderEnvelope struct {
	ID       string `json:"id"`
	CatID    string `json:"catId"`
	Type     string `json:"type"`
	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`
	Time     string `json:"time"`
	State    string `json:"state"`
	Icon     string `json:"icon"`
}

// ReminderInput 创建提醒的输入。
type ReminderInput struct {
	CatID    string `json:"catId"`
	Type     string `json:"type"`
	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`
	Time     string `json:"time"`
	Icon     string `json:"icon"`
	Rule     string `json:"rule"`
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
	rem.UpdatedAt = time.Now().UTC()
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
	if in.Title == "" {
		return nil, errors.InvalidRequest(errors.CodeValidationFailed, "title is required")
	}
	if in.Type == "" {
		in.Type = "custom"
	}
	if in.CatID == "" {
		in.CatID = "both"
	}

	now := time.Now().UTC()
	rem := &model.Reminder{
		Base:      model.Base{ID: model.NewBase(familyID, userID).ID, CreatedBy: userID, CreatedAt: now, UpdatedAt: now},
		FamilyID:  familyID,
		CatID:     in.CatID,
		Type:      in.Type,
		Title:     in.Title,
		Subtitle:  in.Subtitle,
		TimeLabel: in.Time,
		State:     "todo",
		Icon:      in.Icon,
		Rule:      in.Rule,
	}
	if err := s.reminders.Create(ctx, rem); err != nil {
		return nil, errors.Wrap(errors.TypeInternal, errors.CodeInvalidRequest, "failed to create reminder", err)
	}
	return toReminderEnvelope(rem), nil
}

func toReminderEnvelope(r *model.Reminder) *ReminderEnvelope {
	return &ReminderEnvelope{
		ID:       r.ID,
		CatID:    r.CatID,
		Type:     r.Type,
		Title:    r.Title,
		Subtitle: r.Subtitle,
		Time:     r.TimeLabel,
		State:    r.State,
		Icon:     r.Icon,
	}
}
