package app

import (
	"context"
	"encoding/json"
	"time"

	"github.com/meowhome/backend/internal/domain/model"
	"github.com/meowhome/backend/internal/domain/repository"
	"github.com/meowhome/backend/internal/platform/errors"
)

// RecordService 日常记录服务。
type RecordService struct {
	records repository.DailyRecordRepo
	members repository.MemberRepo
	audit   repository.AuditRepo
}

// NewRecordService 创建记录服务。
func NewRecordService(records repository.DailyRecordRepo, members repository.MemberRepo, audit repository.AuditRepo) *RecordService {
	return &RecordService{records: records, members: members, audit: audit}
}

// RecordInput 创建记录的输入（类型专属字段放在 Payload）。
type RecordInput struct {
	Type       string         `json:"type"`
	CatIDs     []string       `json:"cat_ids"`
	OccurredAt string         `json:"occurred_at"`
	Severity   string         `json:"severity"`
	Title      string         `json:"title"`
	Note       string         `json:"note"`
	Source     string         `json:"source"`
	Payload    map[string]any `json:"payload"`
}

// RecordEnvelope 记录响应。
type RecordEnvelope struct {
	ID         string         `json:"id"`
	FamilyID   string         `json:"family_id"`
	CatIDs     []string       `json:"cat_ids"`
	Type       string         `json:"type"`
	OccurredAt string         `json:"occurred_at"`
	Source     string         `json:"source"`
	Severity   string         `json:"severity"`
	Title      string         `json:"title"`
	Note       string         `json:"note"`
	Payload    map[string]any `json:"payload,omitempty"`
}

// 合法记录类型（与前端 mockRecordTypes 对齐）。
var validRecordTypes = map[string]bool{
	"feeding": true, "drinking": true, "elimination": true, "vomit": true,
	"weight": true, "medication": true, "mental": true, "symptom": true,
	"visit": true, "vaccine": true, "deworm": true, "food-change": true,
	"behavior": true, "interaction": true, "photo": true, "milestone": true,
	"custom": true,
}

// Create 创建单条记录。
func (s *RecordService) Create(ctx context.Context, familyID, userID string, in RecordInput) (*RecordEnvelope, error) {
	if err := requireFamilyAccess(ctx, s.members, familyID, userID); err != nil {
		return nil, err
	}
	rec, err := s.build(familyID, userID, in)
	if err != nil {
		return nil, err
	}
	if err := s.records.Create(ctx, rec); err != nil {
		return nil, errors.Wrap(errors.TypeInternal, errors.CodeInvalidRequest, "failed to create record", err)
	}
	s.writeAudit(ctx, familyID, userID, rec.ID)
	return toRecordEnvelope(rec), nil
}

// CreateBatch 批量创建记录（供 AI 确认入库使用）。
func (s *RecordService) CreateBatch(ctx context.Context, familyID, userID string, in []RecordInput) ([]*RecordEnvelope, error) {
	if err := requireFamilyAccess(ctx, s.members, familyID, userID); err != nil {
		return nil, err
	}
	if len(in) == 0 {
		return nil, errors.InvalidRequest(errors.CodeValidationFailed, "records is required")
	}

	out := make([]*RecordEnvelope, 0, len(in))
	for _, item := range in {
		rec, err := s.build(familyID, userID, item)
		if err != nil {
			return nil, err
		}
		if err := s.records.Create(ctx, rec); err != nil {
			return nil, errors.Wrap(errors.TypeInternal, errors.CodeInvalidRequest, "failed to create record", err)
		}
		s.writeAudit(ctx, familyID, userID, rec.ID)
		out = append(out, toRecordEnvelope(rec))
	}
	return out, nil
}

// List 查询家庭记录。
func (s *RecordService) List(ctx context.Context, familyID, userID string, q repository.DailyRecordQuery) ([]*RecordEnvelope, error) {
	if err := requireFamilyAccess(ctx, s.members, familyID, userID); err != nil {
		return nil, err
	}
	q.FamilyID = familyID
	rows, err := s.records.ListByFamily(ctx, q)
	if err != nil {
		return nil, errors.Wrap(errors.TypeInternal, errors.CodeInvalidRequest, "failed to list records", err)
	}
	out := make([]*RecordEnvelope, 0, len(rows))
	for _, r := range rows {
		out = append(out, toRecordEnvelope(r))
	}
	return out, nil
}

func (s *RecordService) build(familyID, userID string, in RecordInput) (*model.DailyRecord, error) {
	if in.Type == "" {
		return nil, errors.InvalidRequest(errors.CodeValidationFailed, "type is required")
	}
	if !validRecordTypes[in.Type] {
		return nil, errors.InvalidRequest(errors.CodeValidationFailed, "unsupported record type: "+in.Type)
	}
	if len(in.CatIDs) == 0 {
		return nil, errors.InvalidRequest(errors.CodeValidationFailed, "cat_ids is required")
	}

	occurred := time.Now().UTC()
	if in.OccurredAt != "" {
		t, err := time.Parse(time.RFC3339, in.OccurredAt)
		if err != nil {
			return nil, errors.InvalidRequest(errors.CodeValidationFailed, "occurred_at must be RFC3339")
		}
		occurred = t.UTC()
	}

	severity := in.Severity
	if severity == "" {
		severity = "normal"
	}
	source := in.Source
	if source == "" {
		source = "manual"
	}

	payload := ""
	if len(in.Payload) > 0 {
		b, err := json.Marshal(in.Payload)
		if err != nil {
			return nil, errors.InvalidRequest(errors.CodeValidationFailed, "payload is not serializable")
		}
		payload = string(b)
	}

	return &model.DailyRecord{
		Base: model.Base{
			ID:        model.NewBase(familyID, userID).ID,
			CreatedBy: userID,
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		},
		FamilyID:   familyID,
		CatIDs:     in.CatIDs,
		RecordType: in.Type,
		OccurredAt: occurred,
		Source:     source,
		Severity:   severity,
		Title:      in.Title,
		Note:       in.Note,
		Payload:    payload,
	}, nil
}

func (s *RecordService) writeAudit(ctx context.Context, familyID, userID, resource string) {
	if s.audit == nil {
		return
	}
	now := time.Now().UTC()
	_ = s.audit.Create(ctx, &model.AuditLog{
		Base:     model.Base{ID: model.NewBase(familyID, userID).ID, CreatedBy: userID, CreatedAt: now, UpdatedAt: now},
		FamilyID: familyID,
		UserID:   userID,
		Action:   "record.create",
		Resource: resource,
	})
}

func toRecordEnvelope(r *model.DailyRecord) *RecordEnvelope {
	env := &RecordEnvelope{
		ID:         r.ID,
		FamilyID:   r.FamilyID,
		CatIDs:     r.CatIDs,
		Type:       r.RecordType,
		OccurredAt: r.OccurredAt.UTC().Format(time.RFC3339),
		Source:     r.Source,
		Severity:   r.Severity,
		Title:      r.Title,
		Note:       r.Note,
	}
	if env.CatIDs == nil {
		env.CatIDs = []string{}
	}
	if r.Payload != "" {
		var p map[string]any
		if err := json.Unmarshal([]byte(r.Payload), &p); err == nil {
			env.Payload = p
		}
	}
	return env
}
