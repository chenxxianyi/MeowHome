package app

import (
	"context"
	"encoding/json"
	"fmt"
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
	records   repository.DailyRecordRepo
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

// SetRecordRepo enables vaccine reminder provenance validation and completion linkage.
func (s *ReminderService) SetRecordRepo(records repository.DailyRecordRepo) {
	s.records = records
}

// FindVaccineReminderBySource returns an active reminder already linked to a vaccine record.
// The caller uses this before validating a retry payload so replaying a confirmed request
// remains idempotent even after its planned date has arrived.
func (s *ReminderService) FindVaccineReminderBySource(ctx context.Context, familyID, userID, recordID string) (*ReminderEnvelope, error) {
	if err := requireFamilyAccess(ctx, s.members, familyID, userID); err != nil {
		return nil, err
	}
	reminder, err := findReminderBySource(ctx, s.reminders, familyID, recordID)
	if err == repository.ErrNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, errors.Wrap(errors.TypeInternal, errors.CodeInvalidRequest, "failed to check vaccine reminder", err)
	}
	if reminder.DeletedAt != nil {
		return nil, nil
	}
	return toReminderEnvelope(reminder), nil
}

// ReminderEnvelope 提醒响应（字段名与前端 Reminder 类型对齐）。
type ReminderEnvelope struct {
	ID                string         `json:"id"`
	CatID             string         `json:"catId"`
	Type              string         `json:"type"`
	Title             string         `json:"title"`
	Subtitle          string         `json:"subtitle"`
	Time              string         `json:"time"`
	State             string         `json:"state"`
	Icon              string         `json:"icon"`
	Rule              string         `json:"rule,omitempty"`
	ScheduledAt       *time.Time     `json:"scheduled_at,omitempty"`
	Timezone          string         `json:"timezone,omitempty"`
	CompletedAt       *time.Time     `json:"completed_at,omitempty"`
	SourceRecordID    string         `json:"source_record_id,omitempty"`
	CompletedRecordID string         `json:"completed_record_id,omitempty"`
	Metadata          map[string]any `json:"metadata,omitempty"`
}

// ReminderInput 创建提醒的输入。
type ReminderInput struct {
	CatID          string         `json:"catId"`
	Type           string         `json:"type"`
	Title          string         `json:"title"`
	Subtitle       string         `json:"subtitle"`
	Time           string         `json:"time"`
	Icon           string         `json:"icon"`
	Rule           string         `json:"rule"`
	ScheduledAt    *time.Time     `json:"scheduled_at,omitempty"`
	Timezone       string         `json:"timezone,omitempty"`
	SourceRecordID string         `json:"-"`
	Metadata       map[string]any `json:"-"`
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
	if rem.Type == "vaccine" {
		return nil, errors.InvalidRequest(errors.CodeValidationFailed, "vaccine reminders must be completed with a linked vaccine record")
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

// CompleteWithRecord completes a vaccine reminder and links the newly saved vaccination record.
func (s *ReminderService) CompleteWithRecord(ctx context.Context, familyID, userID, id, recordID string) (*ReminderEnvelope, error) {
	if err := requireFamilyAccess(ctx, s.members, familyID, userID); err != nil {
		return nil, err
	}
	if s.records == nil {
		return nil, errors.New(errors.TypeInternal, errors.CodeInvalidRequest, "record repository is unavailable")
	}
	rem, err := s.reminders.FindByID(ctx, id)
	if err != nil || rem.FamilyID != familyID || rem.DeletedAt != nil {
		return nil, errors.NotFound(errors.CodeNotFound, "reminder not found")
	}
	if rem.Type != "vaccine" || rem.CatID == "" || rem.CatID == "both" {
		return nil, errors.InvalidRequest(errors.CodeValidationFailed, "only cat-specific vaccine reminders can be linked")
	}
	record, err := s.records.FindByID(ctx, recordID)
	if err != nil || record == nil || record.DeletedAt != nil || record.FamilyID != familyID || record.RecordType != "vaccine" || len(record.CatIDs) != 1 || record.CatIDs[0] != rem.CatID {
		return nil, errors.NotFound(errors.CodeNotFound, "vaccine record not found for reminder cat")
	}
	if rem.Metadata != "" {
		var metadata map[string]any
		if json.Unmarshal([]byte(rem.Metadata), &metadata) == nil {
			reminderVaccineCode := strings.TrimSpace(fmt.Sprint(metadata["vaccine_code"]))
			if reminderVaccineCode != "" {
				var payload map[string]any
				if json.Unmarshal([]byte(record.Payload), &payload) != nil || !strings.EqualFold(reminderVaccineCode, strings.TrimSpace(fmt.Sprint(payload["vaccine_code"]))) {
					return nil, errors.InvalidRequest(errors.CodeValidationFailed, "vaccine record type does not match reminder")
				}
			}
		}
	}
	if rem.State == "done" {
		if rem.CompletedRecordID == recordID {
			return toReminderEnvelope(rem), nil
		}
		return nil, errors.Conflict(errors.CodeConflict, "vaccine reminder was already completed")
	}
	now := time.Now().UTC()
	rem.State = "done"
	rem.UpdatedAt = now
	rem.CompletedAt = &now
	rem.CompletedRecordID = recordID
	if err := s.reminders.Update(ctx, rem); err != nil {
		return nil, errors.Wrap(errors.TypeInternal, errors.CodeInvalidRequest, "failed to complete vaccine reminder", err)
	}
	return toReminderEnvelope(rem), nil
}

// UpdateVaccineSchedule changes the user-confirmed local date for a vaccine reminder.
func (s *ReminderService) UpdateVaccineSchedule(ctx context.Context, familyID, userID, id, scheduledDate, timezone string) (*ReminderEnvelope, error) {
	if err := requireFamilyAccess(ctx, s.members, familyID, userID); err != nil {
		return nil, err
	}
	rem, err := s.reminders.FindByID(ctx, id)
	if err != nil || rem.FamilyID != familyID || rem.DeletedAt != nil {
		return nil, errors.NotFound(errors.CodeNotFound, "reminder not found")
	}
	if rem.Type != "vaccine" || rem.State == "done" {
		return nil, errors.InvalidRequest(errors.CodeValidationFailed, "only pending vaccine reminders can be rescheduled")
	}
	date, err := time.Parse("2006-01-02", strings.TrimSpace(scheduledDate))
	if err != nil {
		return nil, errors.InvalidRequest(errors.CodeValidationFailed, "scheduled_date must use YYYY-MM-DD")
	}
	loc, err := time.LoadLocation(strings.TrimSpace(timezone))
	if err != nil {
		return nil, errors.InvalidRequest(errors.CodeValidationFailed, "invalid timezone")
	}
	scheduledAt := time.Date(date.Year(), date.Month(), date.Day(), 9, 0, 0, 0, loc)
	normalized, err := normalizeReminderInput(ctx, s.cats, s.families, familyID, ReminderInput{
		CatID: rem.CatID, Type: rem.Type, Title: rem.Title, ScheduledAt: &scheduledAt, Timezone: loc.String(),
	}, time.Now().UTC(), true, 255)
	if err != nil {
		return nil, err
	}
	rem.ScheduledAt = normalized.ScheduledAt
	rem.Timezone = normalized.Timezone
	rem.TimeLabel = "09:00"
	if rem.Metadata != "" {
		var metadata map[string]any
		_ = json.Unmarshal([]byte(rem.Metadata), &metadata)
		if metadata == nil {
			metadata = map[string]any{}
		}
		metadata["confirmed_date"] = date.Format("2006-01-02")
		metadata["date_source"] = "user"
		if encoded, marshalErr := json.Marshal(metadata); marshalErr == nil {
			rem.Metadata = string(encoded)
		}
	}
	rem.UpdatedAt = time.Now().UTC()
	if err := s.reminders.Update(ctx, rem); err != nil {
		return nil, errors.Wrap(errors.TypeInternal, errors.CodeInvalidRequest, "failed to reschedule vaccine reminder", err)
	}
	return toReminderEnvelope(rem), nil
}

// CancelVaccineReminder soft-deletes a pending vaccine reminder.
func (s *ReminderService) CancelVaccineReminder(ctx context.Context, familyID, userID, id string) error {
	if err := requireFamilyAccess(ctx, s.members, familyID, userID); err != nil {
		return err
	}
	rem, err := s.reminders.FindByID(ctx, id)
	if err != nil || rem.FamilyID != familyID || rem.DeletedAt != nil {
		return errors.NotFound(errors.CodeNotFound, "reminder not found")
	}
	if rem.Type != "vaccine" || rem.State == "done" {
		return errors.InvalidRequest(errors.CodeValidationFailed, "only pending vaccine reminders can be canceled")
	}
	if err := s.reminders.Delete(ctx, id); err != nil {
		return errors.Wrap(errors.TypeInternal, errors.CodeInvalidRequest, "failed to cancel vaccine reminder", err)
	}
	return nil
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
	metadataJSON := ""
	if in.SourceRecordID != "" {
		if in.Type != "vaccine" || s.records == nil {
			return nil, errors.InvalidRequest(errors.CodeValidationFailed, "vaccine source record is invalid")
		}
		record, recordErr := s.records.FindByID(ctx, in.SourceRecordID)
		if recordErr != nil || record == nil || record.DeletedAt != nil || record.FamilyID != familyID || record.RecordType != "vaccine" || len(record.CatIDs) != 1 || record.CatIDs[0] != in.CatID {
			return nil, errors.NotFound(errors.CodeNotFound, "vaccine source record not found")
		}
		if _, ok := s.reminders.(repository.ReminderSourceRepo); !ok {
			return nil, errors.New(errors.TypeInternal, errors.CodeInvalidRequest, "idempotent vaccine reminders are unavailable")
		}
		if len(in.Metadata) > 0 {
			encoded, marshalErr := json.Marshal(in.Metadata)
			if marshalErr != nil {
				return nil, errors.InvalidRequest(errors.CodeValidationFailed, "reminder metadata is invalid")
			}
			metadataJSON = string(encoded)
		}
		if existing, lookupErr := findReminderBySource(ctx, s.reminders, familyID, in.SourceRecordID); lookupErr == nil {
			if existing.DeletedAt == nil {
				return toReminderEnvelope(existing), nil
			}
			existing.DeletedAt = nil
			existing.CatID = in.CatID
			existing.Type = in.Type
			existing.Title = in.Title
			existing.Subtitle = in.Subtitle
			existing.TimeLabel = in.Time
			existing.State = "todo"
			existing.Icon = in.Icon
			existing.Rule = in.Rule
			existing.ScheduledAt = in.ScheduledAt
			existing.Timezone = in.Timezone
			existing.CompletedAt = nil
			existing.CompletedRecordID = ""
			existing.Metadata = metadataJSON
			existing.UpdatedAt = now
			if updateErr := s.reminders.Update(ctx, existing); updateErr != nil {
				return nil, errors.Wrap(errors.TypeInternal, errors.CodeInvalidRequest, "failed to restore vaccine reminder", updateErr)
			}
			return toReminderEnvelope(existing), nil
		} else if lookupErr != repository.ErrNotFound {
			return nil, errors.Wrap(errors.TypeInternal, errors.CodeInvalidRequest, "failed to check vaccine reminder", lookupErr)
		}
	}
	rem := &model.Reminder{
		Base:           model.Base{ID: model.NewBase(familyID, userID).ID, CreatedBy: userID, CreatedAt: now, UpdatedAt: now},
		FamilyID:       familyID,
		CatID:          in.CatID,
		Type:           in.Type,
		Title:          in.Title,
		Subtitle:       in.Subtitle,
		TimeLabel:      in.Time,
		State:          "todo",
		Icon:           in.Icon,
		Rule:           in.Rule,
		ScheduledAt:    in.ScheduledAt,
		Timezone:       in.Timezone,
		SourceRecordID: optionalString(in.SourceRecordID),
		Metadata:       metadataJSON,
	}
	if err := s.reminders.Create(ctx, rem); err != nil {
		if in.SourceRecordID != "" {
			if existing, lookupErr := findReminderBySource(ctx, s.reminders, familyID, in.SourceRecordID); lookupErr == nil {
				return toReminderEnvelope(existing), nil
			}
		}
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
	sourceRecordID := ""
	if r.SourceRecordID != nil {
		sourceRecordID = *r.SourceRecordID
	}
	env := &ReminderEnvelope{
		ID:                r.ID,
		CatID:             r.CatID,
		Type:              r.Type,
		Title:             r.Title,
		Subtitle:          r.Subtitle,
		Time:              r.TimeLabel,
		State:             r.State,
		Icon:              r.Icon,
		Rule:              r.Rule,
		ScheduledAt:       r.ScheduledAt,
		Timezone:          r.Timezone,
		CompletedAt:       r.CompletedAt,
		SourceRecordID:    sourceRecordID,
		CompletedRecordID: r.CompletedRecordID,
	}
	if r.Metadata != "" {
		var metadata map[string]any
		if json.Unmarshal([]byte(r.Metadata), &metadata) == nil {
			env.Metadata = metadata
		}
	}
	return env
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func findReminderBySource(ctx context.Context, reminders repository.ReminderRepo, familyID, recordID string) (*model.Reminder, error) {
	repo, ok := reminders.(repository.ReminderSourceRepo)
	if !ok {
		return nil, errors.New(errors.TypeInternal, errors.CodeInvalidRequest, "reminder source lookup is unavailable")
	}
	return repo.FindBySourceRecord(ctx, familyID, recordID)
}
