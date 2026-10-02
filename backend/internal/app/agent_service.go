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
	repo           repository.AgentRepo
	records        repository.DailyRecordRepo
	reminders      repository.ReminderRepo
	cats           repository.CatRepo
	families       repository.FamilyRepo
	members        repository.MemberRepo
	audit          repository.AuditRepo
	llm            LLMProvider
	healthProfiles repository.CatHealthProfileRepo
	enabled        bool
	clock          func() time.Time
}

func NewAgentService(repo repository.AgentRepo, records repository.DailyRecordRepo, reminders repository.ReminderRepo, cats repository.CatRepo, families repository.FamilyRepo, members repository.MemberRepo, enabled bool) *AgentService {
	return &AgentService{repo: repo, records: records, reminders: reminders, cats: cats, families: families, members: members, enabled: enabled, clock: func() time.Time { return time.Now().UTC() }}
}

func (s *AgentService) SetAuditRepo(audit repository.AuditRepo) { s.audit = audit }
func (s *AgentService) SetLLMProvider(provider LLMProvider)     { s.llm = provider }

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
	m, err := s.repo.FindMessage(ctx, familyID, messageID)
	if err != nil && err != repository.ErrNotFound {
		return nil, err
	}
	if err != nil || m == nil || m.FamilyID != familyID || !s.canReadMessage(ctx, familyID, userID, m) {
		return nil, apperr.NotFound(apperr.CodeNotFound, "agent message not found")
	}
	return toAgentMessage(m), nil
}

func (s *AgentService) canReadMessage(ctx context.Context, familyID, userID string, m *model.AgentMessage) bool {
	if m.Role == string(RoleTool) {
		return false
	}
	if m.Visibility == string(VisibilityFamily) {
		return true
	}
	if m.Visibility != string(VisibilityPrivate) || m.SessionID == "" {
		return false
	}
	session, err := s.repo.FindSession(ctx, familyID, userID, m.SessionID)
	return err == nil && session.FamilyID == familyID && session.UserID == userID
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
	m, err := s.repo.FindMessage(ctx, familyID, messageID)
	if err != nil && err != repository.ErrNotFound {
		return nil, err
	}
	if err != nil || m == nil || m.FamilyID != familyID || m.Visibility != string(VisibilityFamily) {
		return nil, apperr.NotFound(apperr.CodeNotFound, "agent message not found")
	}
	if m.Visibility == string(VisibilityPrivate) && m.UserID != userID {
		return nil, apperr.NotFound(apperr.CodeNotFound, "agent message not found")
	}
	if m.ActionStatus == DraftConfirmed {
		return nil, apperr.Conflict(CodeAgentConflict, "confirmed draft cannot be edited")
	}
	if m.DisplayStatus == DraftDismissed || m.ActionStatus == DraftDismissed || m.ActionStatus == DraftExpired {
		return nil, apperr.Conflict(CodeAgentConflict, "dismissed or expired draft cannot be edited")
	}
	if m.ActionStatus == DraftPending && m.DraftExpiresAt != nil && !m.DraftExpiresAt.After(s.clock()) {
		return nil, apperr.Conflict(CodeAgentDraftExpired, "draft expired")
	}
	if in.ExpectedVersion != m.DraftVersion {
		return nil, apperr.Conflict(CodeAgentConflict, "draft version conflict")
	}
	if err := s.validateReminder(ctx, familyID, in.Reminder); err != nil {
		return nil, err
	}
	oldStatus := m.ActionStatus
	m.DraftVersion++
	m.ActionStatus = DraftPending
	m.Type = AgentTypeReminderDraft
	m.DraftPayload = encodeJSON(in.Reminder)
	expires := s.clock().Add(AgentDraftDefaultTTL)
	m.DraftExpiresAt = &expires
	if err := s.repo.UpdateMessageIfVersion(ctx, familyID, m, in.ExpectedVersion, oldStatus); err != nil {
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
	m, err := s.repo.FindMessage(ctx, familyID, messageID)
	if err != nil && err != repository.ErrNotFound {
		return nil, err
	}
	if err != nil || m == nil || m.FamilyID != familyID || m.Visibility != string(VisibilityFamily) {
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
	if in.ScheduledAt == nil {
		return nil, apperr.InvalidRequest(apperr.CodeValidationFailed, "scheduled_at is required before confirmation")
	}
	now := s.clock().UTC()
	rem := &model.Reminder{Base: model.Base{ID: id.ULIDGenerator{}.New(), CreatedBy: userID, CreatedAt: now, UpdatedAt: now}, FamilyID: familyID, CatID: in.CatID, Type: in.Type, Title: in.Title, Subtitle: in.Subtitle, TimeLabel: in.Time, ScheduledAt: in.ScheduledAt, Timezone: in.Timezone, State: "todo", Icon: in.Icon, Rule: in.Rule}
	created, err := s.repo.ConfirmMessageAndCreateReminder(ctx, familyID, userID, m.ID, expected, now, rem)
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, apperr.NotFound(apperr.CodeNotFound, "agent draft no longer available")
		}
		if err == repository.ErrConflict {
			latest, findErr := s.repo.FindMessage(ctx, familyID, m.ID)
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
	m, err := s.repo.FindMessage(ctx, familyID, messageID)
	if err != nil && err != repository.ErrNotFound {
		return err
	}
	if err != nil || m == nil || m.FamilyID != familyID {
		return apperr.NotFound(apperr.CodeNotFound, "agent message not found")
	}
	if !s.canReadMessage(ctx, familyID, userID, m) {
		return apperr.NotFound(apperr.CodeNotFound, "agent message not found")
	}
	if m.DisplayStatus != DraftDismissed {
		expectedVersion := m.DraftVersion
		expectedStatus := m.ActionStatus
		m.DisplayStatus = DraftDismissed
		if m.ActionStatus == DraftPending {
			m.ActionStatus = DraftDismissed
		}
		if err := s.repo.UpdateMessageIfVersion(ctx, familyID, m, expectedVersion, expectedStatus); err != nil {
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
	in.Message = strings.TrimSpace(in.Message)
	if in.Message == "" || len([]rune(in.Message)) > AgentChatMessageMaxLen {
		return nil, apperr.InvalidRequest(apperr.CodeValidationFailed, "message length is invalid")
	}
	if in.ClientMessageID == "" || len(in.ClientMessageID) > 128 {
		return nil, apperr.InvalidRequest(apperr.CodeValidationFailed, "client_message_id length is invalid")
	}
	var session *model.AgentSession
	var err error
	var existingMessage *model.AgentMessage
	if in.ClientMessageID != "" {
		if old, e := s.repo.FindUserMessageByClientID(ctx, familyID, userID, in.ClientMessageID); e == nil {
			existingMessage = old
			if old.Role == string(RoleUser) && old.Body != in.Message {
				return nil, apperr.Conflict(CodeAgentConflict, "client_message_id was already used for different content")
			}
			if in.SessionID != "" && old.SessionID != in.SessionID {
				return nil, apperr.Conflict(CodeAgentConflict, "client_message_id belongs to another session")
			}
			session, err = s.repo.FindSession(ctx, familyID, userID, old.SessionID)
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
		session, err = s.repo.FindSession(ctx, familyID, userID, in.SessionID)
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
		if old, e := s.repo.FindMessageByClientID(ctx, familyID, userID, session.ID, in.ClientMessageID); e == nil {
			return &AgentChatResponse{SessionID: session.ID, Message: toAgentMessage(old), Degraded: true}, nil
		}
	}
	now := s.clock().UTC()
	userMsg := &model.AgentMessage{Base: model.Base{ID: id.ULIDGenerator{}.New(), CreatedBy: userID, CreatedAt: now, UpdatedAt: now}, FamilyID: familyID, SessionID: session.ID, UserID: userID, Role: string(RoleUser), Visibility: string(VisibilityPrivate), Type: AgentTypeChatAnswer, Severity: "info", Title: "用户消息", Body: in.Message, GeneratedAt: now, Model: "chat-llm-v1", ClientMessageID: &in.ClientMessageID}
	if existingMessage == nil || existingMessage.Role != string(RoleUser) {
		if err := s.repo.CreateMessage(ctx, userMsg); err != nil && err != repository.ErrDuplicateKey {
			return nil, err
		}
	}
	assistant := &model.AgentMessage{Base: model.Base{ID: id.ULIDGenerator{}.New(), CreatedBy: "system", CreatedAt: now, UpdatedAt: now}, FamilyID: familyID, SessionID: session.ID, UserID: userID, Role: string(RoleAssistant), Visibility: string(VisibilityPrivate), Type: AgentTypeChatAnswer, Severity: "info", Title: "猫管家", Body: "我可以帮你查询本家庭的猫咪记录、趋势和待办。请告诉我猫咪名称或具体时间范围；如果信息不明确，我会先请你澄清。", GeneratedAt: now, Model: "chat-llm-v1", Disclaimer: "当前为确定性降级回复，未调用模型。", ClientMessageID: &in.ClientMessageID}
	if err := s.repo.CreateMessage(ctx, assistant); err != nil {
		if err == repository.ErrDuplicateKey && in.ClientMessageID != "" {
			if old, findErr := s.repo.FindMessageByClientID(ctx, familyID, userID, session.ID, in.ClientMessageID); findErr == nil {
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
			last := rows[limit-1]
			out.NextCursor = last.UpdatedAt.UTC().Format(time.RFC3339Nano) + "|" + last.ID
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
	session, err := s.repo.FindSession(ctx, familyID, userID, sessionID)
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
	rows, err := s.repo.ListMessages(ctx, repository.AgentMessageQuery{FamilyID: familyID, SessionID: sessionID, Visibility: string(VisibilityPrivate), ExcludeTools: true, Before: before, Limit: limit})
	if err != nil {
		return nil, err
	}
	out := &AgentSessionMessagesResponse{Messages: []*AgentMessage{}}
	for i, r := range rows {
		if i >= limit {
			out.NextCursor = cursorFor(rows[limit-1])
			break
		}
		out.Messages = append(out.Messages, toAgentMessage(r))
	}
	return out, nil
}

func (s *AgentService) Patrol(ctx context.Context, familyID, actorID string) (*AgentPatrolResponse, error) {
	result, err := s.patrol(ctx, familyID, actorID, false, s.clock().UTC())
	if err != nil {
		return nil, err
	}
	if s.audit != nil {
		now := s.clock().UTC()
		log := &model.AuditLog{Base: model.Base{ID: id.ULIDGenerator{}.New(), CreatedBy: actorID, CreatedAt: now, UpdatedAt: now}, FamilyID: familyID, UserID: actorID, Action: "agent.patrol.manual", Resource: familyID}
		if err := s.audit.Create(ctx, log); err != nil {
			return nil, apperr.Wrap(apperr.TypeInternal, apperr.CodeInvalidRequest, "failed to audit agent patrol", err)
		}
	}
	return result, nil
}

// PatrolSystem 供进程内调度器使用，限定为系统作用域，不伪造某个用户身份。
func (s *AgentService) PatrolSystem(ctx context.Context, familyID string) (*AgentPatrolResponse, error) {
	return s.PatrolSystemAt(ctx, familyID, s.clock().UTC())
}

// PatrolSystemAt 以原计划时点补跑遗漏窗口，按该窗口的本地日期计算规则。
func (s *AgentService) PatrolSystemAt(ctx context.Context, familyID string, scheduledAt time.Time) (*AgentPatrolResponse, error) {
	return s.patrol(ctx, familyID, "", true, scheduledAt.UTC())
}

// PatrolDangerRecordSystem 为新增 danger 记录生成单条家庭消息；游标重放依靠唯一键去重。
func (s *AgentService) PatrolDangerRecordSystem(ctx context.Context, familyID, recordID string) error {
	if err := s.ensureEnabled(); err != nil {
		return err
	}
	if familyID == "" || recordID == "" {
		return repository.ErrInvalidQuery
	}
	r, err := s.records.FindByID(ctx, recordID)
	if err == repository.ErrNotFound {
		return nil // 扫描后被软删除，不阻塞后续事件
	}
	if err != nil {
		return err
	}
	if r == nil {
		return repository.ErrNotFound
	}
	if r.FamilyID != familyID {
		return repository.ErrNotFound
	}
	if r.DeletedAt != nil || r.Severity != "danger" {
		return nil
	}
	key := fmt.Sprintf("%s:R-01:event:v1:%s", familyID, r.ID)
	if _, err := s.repo.FindMessageByDedup(ctx, familyID, key); err == nil {
		return nil
	} else if err != repository.ErrNotFound {
		return err
	}
	now := s.clock().UTC()
	catID := ""
	if len(r.CatIDs) == 1 {
		catID = r.CatIDs[0]
	}
	m := &model.AgentMessage{Base: model.Base{ID: id.ULIDGenerator{}.New(), CreatedBy: "system", CreatedAt: now, UpdatedAt: now}, FamilyID: familyID, CatID: catID, Role: string(RoleAssistant), Visibility: string(VisibilityFamily), Type: AgentTypePatrolAbnormal, Severity: "danger", Title: "新增高风险记录", Body: "一条新记录标为 danger，请核对原始记录。", Evidence: encodeJSON([]AgentEvidence{{SourceType: "record", SourceID: r.ID, CatIDs: r.CatIDs, OccurredAt: r.OccurredAt, Excerpt: r.Note}}), ActionSuggestions: encodeJSON([]AgentAction{{Type: "view_records", Title: "查看原始记录", CatID: catID, Days: 7}}), GeneratedAt: now, Model: "rule-engine-v1", Disclaimer: "仅根据已记录数据提示，不构成诊断或用药建议。", RuleID: RuleR01, RuleVersion: "1", Scope: "event", WindowStart: &r.CreatedAt, WindowEnd: &now, DedupKey: &key}
	if err := s.repo.CreateMessage(ctx, m); err != nil && err != repository.ErrDuplicateKey {
		return err
	}
	return nil
}

func (s *AgentService) patrol(ctx context.Context, familyID, actorID string, systemScope bool, now time.Time) (*AgentPatrolResponse, error) {
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
	localNow := now.In(loc)
	today := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), 0, 0, 0, 0, loc)
	from := today.AddDate(0, 0, -6).UTC()
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
	// R-04 需要全历史最近一次称重；近七天事件窗口不能代替这个查询。
	for _, cat := range cats {
		if cat.DeletedAt != nil {
			continue
		}
		latest, listErr := s.records.ListByFamily(ctx, repository.DailyRecordQuery{
			FamilyID: familyID, CatID: cat.ID, Types: []string{"weight"},
			To: now.Format(time.RFC3339), Limit: 1,
		})
		if listErr != nil {
			return nil, listErr
		}
		if len(latest) == 1 && latest[0].OccurredAt.Before(from) {
			records = append(records, latest[0])
		}
	}
	reminders, err := s.loadPatrolReminders(ctx, familyID, now, loc)
	if err != nil {
		return nil, err
	}
	candidates := evaluateRules(now, loc, f.CreatedAt, cats, records, reminders)
	out := make([]*AgentMessage, 0, len(candidates))
	for _, c := range candidates {
		if old, e := s.repo.FindMessageByDedup(ctx, familyID, c.DedupKey); e == nil {
			out = append(out, toAgentMessage(old))
			continue
		} else if e != repository.ErrNotFound {
			return nil, e
		}
		generated := now
		dedupKey := c.DedupKey
		messageType := AgentTypePatrolAbnormal
		if c.RuleID == RuleR04 {
			messageType = AgentTypePatrolWeight
		}
		m := &model.AgentMessage{Base: model.Base{ID: id.ULIDGenerator{}.New(), CreatedBy: "system", CreatedAt: now, UpdatedAt: now}, FamilyID: familyID, CatID: c.CatID, Role: string(RoleAssistant), Visibility: string(VisibilityFamily), Type: messageType, Severity: c.Severity, Title: c.Title, Body: c.Body, Evidence: encodeJSON(c.Evidence), ActionSuggestions: encodeJSON(c.Actions), GeneratedAt: generated, Model: "rule-engine-v1", Disclaimer: "仅根据已记录数据提示，不构成诊断或用药建议。", RuleID: c.RuleID, RuleVersion: c.RuleVersion, Scope: c.Scope, WindowStart: &c.WindowStart, WindowEnd: &c.WindowEnd, DedupKey: &dedupKey}
		if err := s.repo.CreatePatrolMessage(ctx, m, today.UTC(), today.AddDate(0, 0, 1).UTC(), 3); err != nil {
			if err == repository.ErrDailyLimit {
				if old, findErr := s.repo.FindMessageByDedup(ctx, familyID, c.DedupKey); findErr == nil {
					out = append(out, toAgentMessage(old))
				} else if findErr != repository.ErrNotFound {
					return nil, findErr
				}
				continue
			}
			if err != repository.ErrDuplicateKey {
				return nil, err
			}
			m, err = s.repo.FindMessageByDedup(ctx, familyID, c.DedupKey)
			if err != nil {
				return nil, err
			}
		}
		out = append(out, toAgentMessage(m))
	}
	return &AgentPatrolResponse{Messages: out}, nil
}

func (s *AgentService) loadPatrolReminders(ctx context.Context, familyID string, now time.Time, loc *time.Location) ([]*model.Reminder, error) {
	overdueBefore := now.Add(-24 * time.Hour)
	dueBefore := now.In(loc).AddDate(0, 0, 7).UTC()
	queries := []repository.ReminderScheduleQuery{
		{FamilyID: familyID, State: "todo", Types: []string{"medication"}, ToExclusive: &overdueBefore, Limit: 500},
		{FamilyID: familyID, State: "todo", Types: []string{"vaccine", "deworm"}, From: &now, ToExclusive: &dueBefore, Limit: 500},
	}
	out := make([]*model.Reminder, 0)
	for _, q := range queries {
		for {
			batch, err := s.reminders.ListScheduled(ctx, q)
			if err != nil {
				return nil, err
			}
			out = append(out, batch...)
			if len(batch) < q.Limit {
				break
			}
			last := batch[len(batch)-1]
			if last.ScheduledAt == nil {
				return nil, repository.ErrInvalidQuery
			}
			q.CursorAt, q.CursorID = last.ScheduledAt, last.ID
		}
	}
	return out, nil
}

func (s *AgentService) validateReminder(ctx context.Context, familyID string, in *AgentReminderInput) error {
	normalized, err := normalizeReminderInput(ctx, s.cats, s.families, familyID, ReminderInput{
		CatID: in.CatID, Type: in.Type, Title: in.Title, Subtitle: in.Subtitle,
		Time: in.Time, ScheduledAt: in.ScheduledAt, Timezone: in.Timezone,
		Icon: in.Icon, Rule: in.Rule,
	}, s.clock().UTC(), true, 200)
	if err != nil {
		return err
	}
	in.CatID, in.Type, in.Title = normalized.CatID, normalized.Type, normalized.Title
	in.ScheduledAt, in.Timezone = normalized.ScheduledAt, normalized.Timezone
	return nil
}

func (s *AgentService) messageList(rows []*model.AgentMessage, limit int) *AgentMessageListResponse {
	out := &AgentMessageListResponse{Messages: []*AgentMessage{}}
	for i, r := range rows {
		if i >= limit {
			out.NextCursor = cursorFor(rows[limit-1])
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
	var draft *AgentReminderInput
	_ = json.Unmarshal([]byte(m.Evidence), &ev)
	_ = json.Unmarshal([]byte(m.ActionSuggestions), &ac)
	if m.DraftPayload != "" {
		var parsed AgentReminderInput
		if json.Unmarshal([]byte(m.DraftPayload), &parsed) == nil {
			draft = &parsed
		}
	}
	return &AgentMessage{ID: m.ID, SessionID: emptyToOmit(m.SessionID), Role: Role(m.Role), Visibility: Visibility(m.Visibility), FamilyID: m.FamilyID, UserID: m.UserID, CatID: m.CatID, Type: m.Type, Severity: m.Severity, Title: m.Title, Body: m.Body, Evidence: ev, Actions: ac, DraftVersion: m.DraftVersion, DraftReminder: draft, DraftExpiresAt: m.DraftExpiresAt, ActionStatus: m.ActionStatus, DisplayStatus: m.DisplayStatus, GeneratedAt: m.GeneratedAt, Model: m.Model, Disclaimer: m.Disclaimer}
}

func toAgentReminderResult(r *model.Reminder) *AgentReminderResult {
	return &AgentReminderResult{ID: r.ID, FamilyID: r.FamilyID, CatID: r.CatID, Type: r.Type, Title: r.Title, Subtitle: r.Subtitle, Time: r.TimeLabel, State: r.State, Icon: r.Icon, Rule: r.Rule, ScheduledAt: r.ScheduledAt, Timezone: r.Timezone, CompletedAt: r.CompletedAt}
}
func emptyToOmit(s string) string { return s }
