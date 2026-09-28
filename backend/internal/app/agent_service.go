package app

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/meowhome/backend/internal/domain/model"
	"github.com/meowhome/backend/internal/domain/repository"
	apperr "github.com/meowhome/backend/internal/platform/errors"
	"github.com/meowhome/backend/internal/platform/id"
)

// AgentService 实现阶段 A 的确定性巡检、消息查询和提醒草稿闭环。
type AgentService struct {
	repo      repository.AgentRepo
	records   repository.DailyRecordRepo
	reminders repository.ReminderRepo
	cats      repository.CatRepo
	families  repository.FamilyRepo
	members   repository.MemberRepo
	enabled   bool
	clock     func() time.Time
}

func NewAgentService(repo repository.AgentRepo, records repository.DailyRecordRepo, reminders repository.ReminderRepo, cats repository.CatRepo, families repository.FamilyRepo, members repository.MemberRepo, enabled bool) *AgentService {
	return &AgentService{repo: repo, records: records, reminders: reminders, cats: cats, families: families, members: members, enabled: enabled, clock: func() time.Time { return time.Now().UTC() }}
}

func (s *AgentService) ensureEnabled() error {
	if !s.enabled {
		return apperr.New(apperr.TypeExternal, CodeAgentDisabled, "agent is disabled")
	}
	return nil
}

func (s *AgentService) access(ctx context.Context, familyID, userID string) error {
	return requireFamilyAccess(ctx, s.members, familyID, userID)
}

func (s *AgentService) ListMessages(ctx context.Context, familyID, userID string, q AgentMessageListQuery) (*AgentMessageListResponse, error) {
	if err := s.access(ctx, familyID, userID); err != nil {
		return nil, err
	}
	if q.Limit <= 0 {
		q.Limit = AgentMessageListDefaultLimit
	}
	if q.Limit > AgentMessageListMaxLimit {
		return nil, apperr.InvalidRequest(apperr.CodeValidationFailed, "limit exceeds maximum")
	}
	if !validAgentType(q.Type) || !validAgentStatus(q.Status) {
		return nil, apperr.InvalidRequest(apperr.CodeValidationFailed, "invalid agent filter")
	}
	if !validAgentCursor(q.Before) {
		return nil, apperr.InvalidRequest(apperr.CodeValidationFailed, "invalid cursor")
	}
	rows, err := s.repo.ListMessages(ctx, repository.AgentMessageQuery{FamilyID: familyID, Type: q.Type, Status: q.Status, Visibility: string(VisibilityFamily), ExcludeDismissed: q.Status == "", Before: q.Before, Limit: q.Limit})
	if err != nil {
		return nil, apperr.Wrap(apperr.TypeInternal, apperr.CodeInvalidRequest, "failed to list agent messages", err)
	}
	return s.messageList(rows, q.Limit), nil
}

func (s *AgentService) GetMessage(ctx context.Context, familyID, userID, messageID string) (*AgentMessage, error) {
	if err := s.access(ctx, familyID, userID); err != nil {
		return nil, err
	}
	m, err := s.repo.FindMessage(ctx, messageID)
	if err != nil || m.FamilyID != familyID || (m.Visibility == string(VisibilityPrivate) && m.UserID != userID) {
		return nil, apperr.NotFound(apperr.CodeNotFound, "agent message not found")
	}
	return toAgentMessage(m), nil
}

func (s *AgentService) EditDraft(ctx context.Context, familyID, userID, messageID string, in AgentReminderEditRequest) (*AgentReminderDraft, error) {
	if err := s.ensureEnabled(); err != nil {
		return nil, err
	}
	if err := s.access(ctx, familyID, userID); err != nil {
		return nil, err
	}
	if in.Reminder == nil || strings.TrimSpace(in.Reminder.Title) == "" {
		return nil, apperr.InvalidRequest(apperr.CodeValidationFailed, "reminder title is required")
	}
	m, err := s.repo.FindMessage(ctx, messageID)
	if err != nil || m.FamilyID != familyID || m.Visibility != string(VisibilityFamily) {
		return nil, apperr.NotFound(apperr.CodeNotFound, "agent message not found")
	}
	if m.Visibility == string(VisibilityPrivate) && m.UserID != userID {
		return nil, apperr.NotFound(apperr.CodeNotFound, "agent message not found")
	}
	if m.ActionStatus == DraftConfirmed {
		return nil, apperr.Conflict(CodeAgentConflict, "confirmed draft cannot be edited")
	}
	if in.ExpectedVersion != m.DraftVersion {
		return nil, apperr.Conflict(CodeAgentConflict, "draft version conflict")
	}
	if err := s.validateReminder(ctx, familyID, in.Reminder); err != nil {
		return nil, err
	}
	m.DraftVersion++
	m.ActionStatus = DraftPending
	m.Type = AgentTypeReminderDraft
	m.DraftPayload = encodeJSON(in.Reminder)
	expires := s.clock().Add(AgentDraftDefaultTTL)
	m.DraftExpiresAt = &expires
	if err := s.repo.UpdateMessageIfVersion(ctx, m, in.ExpectedVersion); err != nil {
		if err == repository.ErrConflict {
			return nil, apperr.Conflict(CodeAgentConflict, "draft version conflict")
		}
		return nil, apperr.Wrap(apperr.TypeInternal, apperr.CodeInvalidRequest, "failed to save reminder draft", err)
	}
	return &AgentReminderDraft{MessageID: m.ID, Version: m.DraftVersion, ExpiresAt: expires, Reminder: in.Reminder, ConfirmedReminderID: m.ConfirmedReminderID}, nil
}

func (s *AgentService) ConfirmDraft(ctx context.Context, familyID, userID, messageID string, expected int) (*AgentConfirmResponse, error) {
	if err := s.ensureEnabled(); err != nil {
		return nil, err
	}
	if err := s.access(ctx, familyID, userID); err != nil {
		return nil, err
	}
	m, err := s.repo.FindMessage(ctx, messageID)
	if err != nil || m.FamilyID != familyID || m.Visibility != string(VisibilityFamily) {
		return nil, apperr.NotFound(apperr.CodeNotFound, "agent message not found")
	}
	if m.ActionStatus == DraftConfirmed && m.ConfirmedReminderID != "" {
		r, e := s.reminders.FindByID(ctx, m.ConfirmedReminderID)
		if e == nil {
			return &AgentConfirmResponse{Reminder: toAgentReminderResult(r), MessageID: m.ID, ActionStatus: DraftConfirmed}, nil
		}
	}
	if expected != m.DraftVersion {
		return nil, apperr.Conflict(CodeAgentConflict, "draft version conflict")
	}
	if m.ActionStatus != DraftPending || m.DraftExpiresAt == nil || !m.DraftExpiresAt.After(s.clock()) {
		return nil, apperr.Conflict(CodeAgentDraftExpired, "draft expired or unavailable")
	}
	var in AgentReminderInput
	if json.Unmarshal([]byte(m.DraftPayload), &in) != nil {
		return nil, apperr.New(apperr.TypeAI, CodeAgentInvalidOut, "invalid reminder draft")
	}
	if err := s.validateReminder(ctx, familyID, &in); err != nil {
		return nil, err
	}
	now := s.clock().UTC()
	rem := &model.Reminder{Base: model.Base{ID: id.ULIDGenerator{}.New(), CreatedBy: userID, CreatedAt: now, UpdatedAt: now}, FamilyID: familyID, CatID: in.CatID, Type: in.Type, Title: in.Title, Subtitle: in.Subtitle, TimeLabel: in.Time, ScheduledAt: in.ScheduledAt, Timezone: in.Timezone, State: "todo", Icon: in.Icon, Rule: in.Rule}
	created, err := s.repo.ConfirmMessageAndCreateReminder(ctx, m.ID, expected, now, rem)
	if err != nil {
		if err == repository.ErrConflict {
			latest, findErr := s.repo.FindMessage(ctx, m.ID)
			if findErr == nil && latest.ActionStatus == DraftConfirmed && latest.ConfirmedReminderID != "" {
				if existing, reminderErr := s.reminders.FindByID(ctx, latest.ConfirmedReminderID); reminderErr == nil {
					return &AgentConfirmResponse{Reminder: toAgentReminderResult(existing), MessageID: latest.ID, ActionStatus: DraftConfirmed}, nil
				}
			}
			return nil, apperr.Conflict(CodeAgentConflict, "draft version conflict")
		}
		return nil, apperr.Wrap(apperr.TypeInternal, apperr.CodeInvalidRequest, "failed to confirm reminder", err)
	}
	if !created {
		return &AgentConfirmResponse{Reminder: toAgentReminderResult(rem), MessageID: m.ID, ActionStatus: DraftConfirmed}, nil
	}
	return &AgentConfirmResponse{Reminder: toAgentReminderResult(rem), MessageID: m.ID, ActionStatus: DraftConfirmed}, nil
}

func (s *AgentService) DismissMessage(ctx context.Context, familyID, userID, messageID string) error {
	if err := s.access(ctx, familyID, userID); err != nil {
		return err
	}
	m, err := s.repo.FindMessage(ctx, messageID)
	if err != nil || m.FamilyID != familyID {
		return apperr.NotFound(apperr.CodeNotFound, "agent message not found")
	}
	if m.Visibility == string(VisibilityPrivate) && m.UserID != userID {
		return apperr.NotFound(apperr.CodeNotFound, "agent message not found")
	}
	if m.ActionStatus == DraftConfirmed {
		return apperr.Conflict(CodeAgentConflict, "confirmed draft cannot be dismissed")
	}
	if m.ActionStatus != DraftDismissed {
		expectedVersion := m.DraftVersion
		m.ActionStatus = DraftDismissed
		if err := s.repo.UpdateMessageIfVersion(ctx, m, expectedVersion); err != nil {
			if err == repository.ErrConflict {
				return apperr.Conflict(CodeAgentConflict, "message state conflict")
			}
			return err
		}
	}
	return nil
}

// Chat 提供无模型时也可用的确定性降级对话，并持久化用户/助手两轮消息。
func (s *AgentService) Chat(ctx context.Context, familyID, userID string, in AgentChatRequest) (*AgentChatResponse, error) {
	if err := s.ensureEnabled(); err != nil {
		return nil, err
	}
	if err := s.access(ctx, familyID, userID); err != nil {
		return nil, err
	}
	if strings.TrimSpace(in.Message) == "" || len([]rune(in.Message)) > AgentChatMessageMaxLen {
		return nil, apperr.InvalidRequest(apperr.CodeValidationFailed, "message length is invalid")
	}
	var session *model.AgentSession
	var err error
	var existingMessage *model.AgentMessage
	if in.ClientMessageID != "" {
		if old, e := s.repo.FindAnyMessageByClientID(ctx, familyID, userID, in.ClientMessageID); e == nil {
			existingMessage = old
			if old.Role == string(RoleUser) && old.Body != in.Message {
				return nil, apperr.Conflict(CodeAgentConflict, "client_message_id was already used for different content")
			}
			if in.SessionID != "" && old.SessionID != in.SessionID {
				return nil, apperr.Conflict(CodeAgentConflict, "client_message_id belongs to another session")
			}
			session, err = s.repo.FindSession(ctx, old.SessionID)
			if err != nil {
				return nil, err
			}
			if old.Role == string(RoleAssistant) {
				return &AgentChatResponse{SessionID: session.ID, Message: toAgentMessage(old), Degraded: true}, nil
			}
		} else if e != repository.ErrNotFound {
			return nil, e
		}
	}
	if in.SessionID != "" {
		session, err = s.repo.FindSession(ctx, in.SessionID)
		if err != nil || session.FamilyID != familyID || session.UserID != userID {
			return nil, apperr.NotFound(apperr.CodeNotFound, "agent session not found")
		}
	} else if session == nil {
		now := s.clock().UTC()
		session = &model.AgentSession{Base: model.Base{ID: id.ULIDGenerator{}.New(), CreatedBy: userID, CreatedAt: now, UpdatedAt: now}, FamilyID: familyID, UserID: userID, Status: "active"}
		if err := s.repo.CreateSession(ctx, session); err != nil {
			return nil, err
		}
	}
	if in.ClientMessageID != "" && session != nil {
		if old, e := s.repo.FindMessageByClientID(ctx, session.ID, in.ClientMessageID); e == nil {
			return &AgentChatResponse{SessionID: session.ID, Message: toAgentMessage(old), Degraded: true}, nil
		}
	}
	now := s.clock().UTC()
	userMsg := &model.AgentMessage{Base: model.Base{ID: id.ULIDGenerator{}.New(), CreatedBy: userID, CreatedAt: now, UpdatedAt: now}, FamilyID: familyID, SessionID: session.ID, UserID: userID, Role: string(RoleUser), Visibility: string(VisibilityPrivate), Type: AgentTypeChatAnswer, Severity: "info", Title: "用户消息", Body: in.Message, GeneratedAt: now, Model: "chat-llm-v1", ClientMessageID: in.ClientMessageID}
	if existingMessage == nil || existingMessage.Role != string(RoleUser) {
		if err := s.repo.CreateMessage(ctx, userMsg); err != nil && err != repository.ErrDuplicateKey {
			return nil, err
		}
	}
	assistant := &model.AgentMessage{Base: model.Base{ID: id.ULIDGenerator{}.New(), CreatedBy: "system", CreatedAt: now, UpdatedAt: now}, FamilyID: familyID, SessionID: session.ID, UserID: userID, Role: string(RoleAssistant), Visibility: string(VisibilityPrivate), Type: AgentTypeChatAnswer, Severity: "info", Title: "猫管家", Body: "我可以帮你查询本家庭的猫咪记录、趋势和待办。请告诉我猫咪名称或具体时间范围；如果信息不明确，我会先请你澄清。", GeneratedAt: now, Model: "chat-llm-v1", Disclaimer: "当前为确定性降级回复，未调用模型。", ClientMessageID: in.ClientMessageID}
	if err := s.repo.CreateMessage(ctx, assistant); err != nil {
		if err == repository.ErrDuplicateKey && in.ClientMessageID != "" {
			if old, findErr := s.repo.FindMessageByClientID(ctx, session.ID, in.ClientMessageID); findErr == nil {
				return &AgentChatResponse{SessionID: session.ID, Message: toAgentMessage(old), Degraded: true}, nil
			}
		}
		return nil, err
	}
	session.LastMessageAt = &now
	session.UpdatedAt = now
	_ = s.repo.UpdateSession(ctx, session)
	return &AgentChatResponse{SessionID: session.ID, Message: toAgentMessage(assistant), Degraded: true}, nil
}

func (s *AgentService) ListSessions(ctx context.Context, familyID, userID, before string, limit int) (*AgentSessionListResponse, error) {
	if err := s.access(ctx, familyID, userID); err != nil {
		return nil, err
	}
	if !validAgentCursor(before) {
		return nil, apperr.InvalidRequest(apperr.CodeValidationFailed, "invalid cursor")
	}
	if limit <= 0 {
		limit = AgentMessageListDefaultLimit
	}
	if limit > AgentMessageListMaxLimit {
		return nil, apperr.InvalidRequest(apperr.CodeValidationFailed, "limit exceeds maximum")
	}
	rows, err := s.repo.ListSessions(ctx, familyID, userID, before, limit)
	if err != nil {
		return nil, err
	}
	out := &AgentSessionListResponse{Sessions: []AgentSessionSummary{}}
	for i, r := range rows {
		if i >= limit {
			out.NextCursor = r.UpdatedAt.UTC().Format(time.RFC3339Nano) + "|" + r.ID
			break
		}
		out.Sessions = append(out.Sessions, AgentSessionSummary{ID: r.ID, Title: r.Title, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt})
	}
	return out, nil
}

func (s *AgentService) ListSessionMessages(ctx context.Context, familyID, userID, sessionID, before string, limit int) (*AgentSessionMessagesResponse, error) {
	if err := s.access(ctx, familyID, userID); err != nil {
		return nil, err
	}
	session, err := s.repo.FindSession(ctx, sessionID)
	if err != nil || session.FamilyID != familyID || session.UserID != userID {
		return nil, apperr.NotFound(apperr.CodeNotFound, "agent session not found")
	}
	if limit <= 0 {
		limit = AgentMessageListDefaultLimit
	}
	if limit > AgentMessageListMaxLimit {
		return nil, apperr.InvalidRequest(apperr.CodeValidationFailed, "limit exceeds maximum")
	}
	if !validAgentCursor(before) {
		return nil, apperr.InvalidRequest(apperr.CodeValidationFailed, "invalid cursor")
	}
	rows, err := s.repo.ListMessages(ctx, repository.AgentMessageQuery{FamilyID: familyID, SessionID: sessionID, Visibility: string(VisibilityPrivate), Before: before, Limit: limit})
	if err != nil {
		return nil, err
	}
	out := &AgentSessionMessagesResponse{Messages: []*AgentMessage{}}
	for i, r := range rows {
		if i >= limit {
			out.NextCursor = cursorFor(r)
			break
		}
		out.Messages = append(out.Messages, toAgentMessage(r))
	}
	return out, nil
}

func (s *AgentService) Patrol(ctx context.Context, familyID, actorID string) (*AgentPatrolResponse, error) {
	return s.patrol(ctx, familyID, actorID, false)
}

// PatrolSystem 供进程内调度器使用，限定为系统作用域，不伪造某个用户身份。
func (s *AgentService) PatrolSystem(ctx context.Context, familyID string) (*AgentPatrolResponse, error) {
	return s.patrol(ctx, familyID, "", true)
}

func (s *AgentService) patrol(ctx context.Context, familyID, actorID string, systemScope bool) (*AgentPatrolResponse, error) {
	if err := s.ensureEnabled(); err != nil {
		return nil, err
	}
	if familyID == "" {
		return nil, apperr.InvalidRequest(apperr.CodeValidationFailed, "family_id is required")
	}
	if !systemScope {
		if role := memberRole(ctx, s.members, familyID, actorID); role != "admin" && role != "owner" {
		return nil, apperr.Forbidden(apperr.CodeFamilyForbidden, "admin role required")
		}
	}
	cats, err := s.cats.ListByFamily(ctx, familyID)
	if err != nil {
		return nil, err
	}
	f, err := s.families.FindByID(ctx, familyID)
	if err != nil {
		return nil, err
	}
	if f.DeletedAt != nil {
		return nil, apperr.NotFound(apperr.CodeNotFound, "family not found")
	}
	loc, err := time.LoadLocation(f.Timezone)
	if err != nil {
		loc = time.FixedZone("CST", 8*3600)
	}
	now := s.clock().UTC()
	from := now.AddDate(0, 0, -15)
	records := make([]*model.DailyRecord, 0)
	var beforeAt *time.Time
	var beforeID string
	for {
		batch, listErr := s.records.ListByFamily(ctx, repository.DailyRecordQuery{FamilyID: familyID, From: from.Format(time.RFC3339), To: now.Format(time.RFC3339), BeforeOccurredAt: beforeAt, BeforeID: beforeID, Limit: 1000})
		if listErr != nil {
			return nil, listErr
		}
		records = append(records, batch...)
		if len(batch) < 1000 {
			break
		}
		last := batch[len(batch)-1]
		cursorAt := last.OccurredAt
		beforeAt, beforeID = &cursorAt, last.ID
	}
	reminders, err := s.reminders.List(ctx, repository.ReminderQuery{FamilyID: familyID})
	if err != nil {
		return nil, err
	}
	candidates := evaluateRules(now, loc, f.CreatedAt, cats, records, reminders)
	out := make([]*AgentMessage, 0, len(candidates))
	nonDangerToday := 0
	if existing, listErr := s.repo.ListMessages(ctx, repository.AgentMessageQuery{FamilyID: familyID, Visibility: string(VisibilityFamily), Limit: AgentMessageListMaxLimit}); listErr == nil {
		localDate := now.In(loc).Format("2006-01-02")
		for _, item := range existing {
			if item.Severity != "danger" && item.GeneratedAt.In(loc).Format("2006-01-02") == localDate {
				nonDangerToday++
			}
		}
	}
	for _, c := range candidates {
		if old, e := s.repo.FindMessageByDedup(ctx, familyID, c.DedupKey); e == nil {
			out = append(out, toAgentMessage(old))
			continue
		}
		if c.Severity != "danger" && nonDangerToday >= 3 {
			continue
		}
		generated := now
		dedupKey := c.DedupKey
		m := &model.AgentMessage{Base: model.Base{ID: id.ULIDGenerator{}.New(), CreatedBy: "system", CreatedAt: now, UpdatedAt: now}, FamilyID: familyID, CatID: c.CatID, Role: string(RoleAssistant), Visibility: string(VisibilityFamily), Type: AgentTypePatrolAbnormal, Severity: c.Severity, Title: c.Title, Body: c.Body, Evidence: encodeJSON(c.Evidence), ActionSuggestions: encodeJSON(c.Actions), GeneratedAt: generated, Model: "rule-engine-v1", Disclaimer: "仅根据已记录数据提示，不构成诊断或用药建议。", RuleID: c.RuleID, RuleVersion: c.RuleVersion, Scope: c.Scope, WindowStart: &c.WindowStart, WindowEnd: &c.WindowEnd, DedupKey: &dedupKey}
		if err := s.repo.CreateMessage(ctx, m); err != nil && err != repository.ErrDuplicateKey {
			return nil, err
		}
		if got, e := s.repo.FindMessageByDedup(ctx, familyID, c.DedupKey); e == nil {
			m = got
		}
		if c.Severity != "danger" {
			nonDangerToday++
		}
		out = append(out, toAgentMessage(m))
	}
	return &AgentPatrolResponse{Messages: out}, nil
}

func (s *AgentService) validateReminder(ctx context.Context, familyID string, in *AgentReminderInput) error {
	if in.Type == "" {
		in.Type = "custom"
	}
	if in.CatID == "" {
		return apperr.InvalidRequest(apperr.CodeValidationFailed, "cat_id is required; use both for a family reminder")
	}
	if in.ScheduledAt != nil && in.ScheduledAt.Location() != time.UTC {
		*in.ScheduledAt = in.ScheduledAt.UTC()
	}
	if in.ScheduledAt != nil && in.ScheduledAt.Before(s.clock().Add(-time.Minute)) {
		return apperr.InvalidRequest(apperr.CodeValidationFailed, "scheduled_at must be in the future")
	}
	if in.Timezone == "" {
		if f, err := s.families.FindByID(ctx, familyID); err == nil {
			in.Timezone = f.Timezone
		}
	}
	if in.Timezone != "" {
		if _, err := time.LoadLocation(in.Timezone); err != nil {
			return apperr.InvalidRequest(apperr.CodeValidationFailed, "invalid timezone")
		}
	}
	if in.CatID != "both" {
		c, err := s.cats.FindByID(ctx, in.CatID)
		if err != nil || c.FamilyID != familyID || c.DeletedAt != nil {
			return apperr.Forbidden(apperr.CodeFamilyForbidden, "cat does not belong to family")
		}
	}
	return nil
}

func (s *AgentService) messageList(rows []*model.AgentMessage, limit int) *AgentMessageListResponse {
	out := &AgentMessageListResponse{Messages: []*AgentMessage{}}
	for i, r := range rows {
		if i >= limit {
			out.NextCursor = cursorFor(r)
			break
		}
		out.Messages = append(out.Messages, toAgentMessage(r))
	}
	return out
}
func cursorFor(m *model.AgentMessage) string {
	return m.GeneratedAt.UTC().Format(time.RFC3339Nano) + "|" + m.ID
}
func validAgentType(v string) bool {
	return v == "" || v == AgentTypePatrolAbnormal || v == AgentTypePatrolWeight || v == AgentTypeChatAnswer || v == AgentTypeReminderDraft
}
func validAgentStatus(v string) bool {
	return v == "" || v == DraftPending || v == DraftConfirmed || v == DraftDismissed || v == DraftExpired
}
func validAgentCursor(v string) bool {
	if v == "" {
		return true
	}
	p := strings.SplitN(v, "|", 2)
	if len(p) != 2 || p[1] == "" {
		return false
	}
	_, err := time.Parse(time.RFC3339Nano, p[0])
	return err == nil
}
func toAgentMessage(m *model.AgentMessage) *AgentMessage {
	var ev []AgentEvidence
	var ac []AgentAction
	_ = json.Unmarshal([]byte(m.Evidence), &ev)
	_ = json.Unmarshal([]byte(m.ActionSuggestions), &ac)
	return &AgentMessage{ID: m.ID, SessionID: emptyToOmit(m.SessionID), Role: Role(m.Role), Visibility: Visibility(m.Visibility), FamilyID: m.FamilyID, UserID: m.UserID, CatID: m.CatID, Type: m.Type, Severity: m.Severity, Title: m.Title, Body: m.Body, Evidence: ev, Actions: ac, DraftVersion: m.DraftVersion, ActionStatus: m.ActionStatus, GeneratedAt: m.GeneratedAt, Model: m.Model, Disclaimer: m.Disclaimer}
}

func toAgentReminderResult(r *model.Reminder) *AgentReminderResult {
	return &AgentReminderResult{ID: r.ID, FamilyID: r.FamilyID, CatID: r.CatID, Type: r.Type, Title: r.Title, Subtitle: r.Subtitle, Time: r.TimeLabel, State: r.State, Icon: r.Icon, Rule: r.Rule, ScheduledAt: r.ScheduledAt, Timezone: r.Timezone, CompletedAt: r.CompletedAt}
}
func emptyToOmit(s string) string { return s }

// Keep fmt imported by older Go formatters when generated messages are extended.
var _ = fmt.Sprintf
