package app

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/meowhome/backend/internal/domain/model"
	"github.com/meowhome/backend/internal/domain/repository"
	apperr "github.com/meowhome/backend/internal/platform/errors"
	"go.uber.org/zap"
)

// AgentToolScope 只能由鉴权后的服务端构造，不能从模型参数读取。
type AgentToolScope struct {
	FamilyID             string
	UserID               string
	SessionID            string
	TurnID               string
	DraftTargetMessageID string
	ExpectedDraftVersion int
}

type AgentToolResult struct {
	Content   json.RawMessage
	Evidence  []AgentEvidence
	Draft     *AgentReminderDraft
	Truncated bool
}

func AgentToolDefinitions() []ToolDef {
	return []ToolDef{
		{Name: "listRecords", Description: "读取指定猫咪最近 1–90 天的原始记录，最多返回 50 条", Parameters: json.RawMessage(`{"type":"object","additionalProperties":false,"properties":{"cat_id":{"type":"string"},"type":{"type":"string"},"days":{"type":"integer","minimum":1,"maximum":90},"limit":{"type":"integer","minimum":1,"maximum":50}},"required":["cat_id"]}`)},
		{Name: "getCatProfile", Description: "读取指定猫咪的基本信息及已登记健康档案", Parameters: json.RawMessage(`{"type":"object","additionalProperties":false,"properties":{"cat_id":{"type":"string"}},"required":["cat_id"]}`)},
		{Name: "getTrends", Description: "读取指定猫咪 1–90 天的已记录趋势", Parameters: json.RawMessage(`{"type":"object","additionalProperties":false,"properties":{"cat_id":{"type":"string"},"metric":{"type":"string","enum":["weight","feeding","drinking","vomit"]},"days":{"type":"integer","minimum":1,"maximum":90}},"required":["cat_id","metric","days"]}`)},
		{Name: "createReminderDraft", Description: "保存待用户确认的提醒草稿，不创建正式提醒", Parameters: json.RawMessage(`{"type":"object","additionalProperties":false,"properties":{"cat_id":{"type":"string"},"type":{"type":"string"},"title":{"type":"string","minLength":1,"maxLength":120},"scheduled_at":{"type":"string","format":"date-time"},"timezone":{"type":"string"}},"required":["cat_id","title"]}`)},
	}
}

func decodeToolArgs(raw string, dst any) error {
	if len(raw) == 0 || len(raw) > 16<<10 {
		return apperr.InvalidRequest(apperr.CodeValidationFailed, "tool arguments length is invalid")
	}
	decoder := json.NewDecoder(bytes.NewBufferString(raw))
	decoder.DisallowUnknownFields()
	if decoder.Decode(dst) != nil {
		return apperr.InvalidRequest(apperr.CodeValidationFailed, "invalid tool arguments")
	}
	var extra any
	if decoder.Decode(&extra) != io.EOF {
		return apperr.InvalidRequest(apperr.CodeValidationFailed, "invalid tool arguments")
	}
	return nil
}

func (s *AgentService) SetHealthProfileRepo(repo repository.CatHealthProfileRepo) {
	s.healthProfiles = repo
}

func (s *AgentService) ToolCatCandidates(ctx context.Context, scope AgentToolScope) ([]map[string]string, error) {
	if err := s.access(ctx, scope.FamilyID, scope.UserID); err != nil {
		return nil, err
	}
	cats, err := s.cats.ListByFamily(ctx, scope.FamilyID)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]string, 0, len(cats))
	for _, cat := range cats {
		if cat != nil && cat.FamilyID == scope.FamilyID && cat.DeletedAt == nil {
			out = append(out, map[string]string{"id": cat.ID, "name": cat.Name})
		}
	}
	return out, nil
}

func (s *AgentService) toolCat(ctx context.Context, familyID, catID string) (*model.Cat, error) {
	if catID == "" {
		return nil, apperr.InvalidRequest(apperr.CodeValidationFailed, "cat_id is required")
	}
	cat, err := s.cats.FindByID(ctx, catID)
	if err != nil && err != repository.ErrNotFound {
		return nil, err
	}
	if err == repository.ErrNotFound || cat == nil || cat.FamilyID != familyID || cat.DeletedAt != nil {
		return nil, apperr.NotFound(apperr.CodeNotFound, "cat not found")
	}
	return cat, nil
}

func (s *AgentService) ExecuteTool(ctx context.Context, scope AgentToolScope, name, rawArgs string) (result *AgentToolResult, err error) {
	toolName := "unknown"
	switch name {
	case "listRecords", "getCatProfile", "getTrends", "createReminderDraft":
		toolName = name
	}
	defer func() {
		s.logger.Info("agent_tool_finished", zap.String("request_id", agentRequestID(ctx)), zap.String("turn_id", scope.TurnID), zap.String("family_id", scope.FamilyID), zap.String("tool", toolName), zap.Bool("success", err == nil), zap.String("reason", agentRunErrorCode(err)))
	}()
	if err := s.ensureEnabled(scope.FamilyID); err != nil {
		return nil, err
	}
	if err := s.access(ctx, scope.FamilyID, scope.UserID); err != nil {
		return nil, err
	}
	switch name {
	case "listRecords":
		return s.toolListRecords(ctx, scope, rawArgs)
	case "getCatProfile":
		return s.toolGetCatProfile(ctx, scope, rawArgs)
	case "getTrends":
		return s.toolGetTrends(ctx, scope, rawArgs)
	case "createReminderDraft":
		return s.toolCreateReminderDraft(ctx, scope, rawArgs)
	default:
		return nil, apperr.InvalidRequest(apperr.CodeValidationFailed, "unknown agent tool")
	}
}

func (s *AgentService) toolLocation(ctx context.Context, familyID string) (*time.Location, error) {
	family, err := s.families.FindByID(ctx, familyID)
	if err != nil {
		return nil, err
	}
	if family == nil || family.DeletedAt != nil {
		return nil, repository.ErrNotFound
	}
	loc, err := time.LoadLocation(family.Timezone)
	if err != nil {
		return nil, apperr.InvalidRequest(apperr.CodeValidationFailed, "invalid family timezone")
	}
	return loc, nil
}

func toolResult(value any, evidence []AgentEvidence, truncated bool) (*AgentToolResult, error) {
	body, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return &AgentToolResult{Content: body, Evidence: evidence, Truncated: truncated}, nil
}

func toolExcerpt(value string, limit int) string {
	runes := []rune(strings.TrimSpace(value))
	if len(runes) > limit {
		return string(runes[:limit])
	}
	return string(runes)
}

func recordHasCat(record *model.DailyRecord, catID string) bool {
	for _, id := range record.CatIDs {
		if id == catID {
			return true
		}
	}
	return false
}

func (s *AgentService) toolListRecords(ctx context.Context, scope AgentToolScope, raw string) (*AgentToolResult, error) {
	var args struct {
		CatID string `json:"cat_id"`
		Type  string `json:"type"`
		Days  *int   `json:"days"`
		Limit *int   `json:"limit"`
	}
	if err := decodeToolArgs(raw, &args); err != nil {
		return nil, err
	}
	if _, err := s.toolCat(ctx, scope.FamilyID, args.CatID); err != nil {
		return nil, err
	}
	days, limit := 7, 20
	if args.Days != nil {
		days = *args.Days
	}
	if args.Limit != nil {
		limit = *args.Limit
	}
	if days < 1 || days > 90 || limit < 1 || limit > 50 {
		return nil, apperr.InvalidRequest(apperr.CodeValidationFailed, "tool range is invalid")
	}
	if args.Type != "" && !validRecordTypes[args.Type] {
		return nil, apperr.InvalidRequest(apperr.CodeValidationFailed, "record type is invalid")
	}
	loc, err := s.toolLocation(ctx, scope.FamilyID)
	if err != nil {
		return nil, err
	}
	now := s.clock().UTC()
	local := now.In(loc)
	today := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
	from := today.AddDate(0, 0, -(days - 1))
	q := repository.DailyRecordQuery{FamilyID: scope.FamilyID, CatID: args.CatID, From: from.UTC().Format(time.RFC3339), To: now.Format(time.RFC3339), Limit: limit + 1}
	if args.Type != "" {
		q.Types = []string{args.Type}
	}
	rows, err := s.records.ListByFamily(ctx, q)
	if err != nil {
		return nil, err
	}
	truncated := len(rows) > limit
	if truncated {
		rows = rows[:limit]
	}
	type recordOut struct {
		ID         string    `json:"id"`
		CatIDs     []string  `json:"cat_ids"`
		Type       string    `json:"type"`
		OccurredAt time.Time `json:"occurred_at"`
		Severity   string    `json:"severity"`
		Title      string    `json:"title"`
		Excerpt    string    `json:"excerpt"`
	}
	out := make([]recordOut, 0, len(rows))
	evidence := make([]AgentEvidence, 0, len(rows))
	for _, row := range rows {
		if row == nil || row.FamilyID != scope.FamilyID || row.DeletedAt != nil || !recordHasCat(row, args.CatID) || (args.Type != "" && row.RecordType != args.Type) || row.OccurredAt.Before(from) || row.OccurredAt.After(now) {
			return nil, repository.ErrInvalidQuery
		}
		excerpt := toolExcerpt(row.Note, 300)
		out = append(out, recordOut{row.ID, row.CatIDs, row.RecordType, row.OccurredAt, row.Severity, row.Title, excerpt})
		evidence = append(evidence, evidenceForRecord(row, loc, excerpt))
	}
	return toolResult(map[string]any{"records": out, "truncated": truncated, "window_start": from.UTC(), "window_end": now}, evidence, truncated)
}

func (s *AgentService) toolGetCatProfile(ctx context.Context, scope AgentToolScope, raw string) (*AgentToolResult, error) {
	var args struct {
		CatID string `json:"cat_id"`
	}
	if err := decodeToolArgs(raw, &args); err != nil {
		return nil, err
	}
	cat, err := s.toolCat(ctx, scope.FamilyID, args.CatID)
	if err != nil {
		return nil, err
	}
	if s.healthProfiles == nil {
		return nil, apperr.New(apperr.TypeExternal, CodeAgentUnavailable, "health profile repository unavailable")
	}
	profile, err := s.healthProfiles.FindByCatID(ctx, cat.ID)
	recorded := true
	if err == repository.ErrNotFound {
		recorded = false
	} else if err != nil {
		return nil, err
	} else if profile == nil || profile.CatID != cat.ID || profile.DeletedAt != nil {
		return nil, repository.ErrInvalidQuery
	}
	type healthOut struct {
		Diseases          []string `json:"diseases,omitempty"`
		Allergies         []string `json:"allergies,omitempty"`
		Contraindications []string `json:"contraindications,omitempty"`
	}
	health := healthOut{}
	if recorded && profile != nil {
		if profile.Diseases != "" && json.Unmarshal([]byte(profile.Diseases), &health.Diseases) != nil {
			return nil, repository.ErrInvalidQuery
		}
		if profile.Allergies != "" && json.Unmarshal([]byte(profile.Allergies), &health.Allergies) != nil {
			return nil, repository.ErrInvalidQuery
		}
		if profile.Contraindications != "" && json.Unmarshal([]byte(profile.Contraindications), &health.Contraindications) != nil {
			return nil, repository.ErrInvalidQuery
		}
	}
	return toolResult(map[string]any{"id": cat.ID, "name": cat.Name, "gender": cat.Gender, "breed": cat.Breed, "birthday": cat.Birthday, "neutered": cat.Neutered, "profile_recorded": recorded, "health": health}, []AgentEvidence{{SourceType: "cat_profile", SourceID: cat.ID, CatIDs: []string{cat.ID}, OccurredAt: cat.CreatedAt, Excerpt: "猫咪档案"}}, false)
}
