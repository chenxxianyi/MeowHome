package mysql

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/meowhome/backend/internal/domain/model"
	"github.com/meowhome/backend/internal/domain/repository"
	"github.com/meowhome/backend/internal/platform/id"
)

type AgentRepo struct{ db *gorm.DB }

func NewAgentRepo(db *gorm.DB) *AgentRepo { return &AgentRepo{db: db} }

var _ repository.AgentRepo = (*AgentRepo)(nil)

func (r *AgentRepo) CreateSession(ctx context.Context, s *model.AgentSession) error {
	return r.db.WithContext(ctx).Create(s).Error
}

func (r *AgentRepo) UpdateSession(ctx context.Context, s *model.AgentSession) error {
	result := r.db.WithContext(ctx).Model(&model.AgentSession{}).
		Where("id = ? AND family_id = ? AND user_id = ? AND deleted_at IS NULL", s.ID, s.FamilyID, s.UserID).
		Updates(map[string]any{"title": s.Title, "status": s.Status, "last_message_at": s.LastMessageAt, "updated_at": s.UpdatedAt})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return repository.ErrNotFound
	}
	return nil
}

func (r *AgentRepo) FindSession(ctx context.Context, familyID, userID, id string) (*model.AgentSession, error) {
	var s model.AgentSession
	err := r.db.WithContext(ctx).Where("id = ? AND family_id = ? AND user_id = ? AND deleted_at IS NULL", id, familyID, userID).First(&s).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, repository.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *AgentRepo) ListSessions(ctx context.Context, familyID, userID, before string, limit int) ([]*model.AgentSession, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	tx := r.db.WithContext(ctx).Where("family_id = ? AND user_id = ? AND deleted_at IS NULL", familyID, userID)
	if t, err := parseAgentCursor(before); err == nil && !t.IsZero() {
		tx = tx.Where("(updated_at, id) < (?, ?)", t, beforeCursorID(before))
	}
	var out []*model.AgentSession
	err := tx.Order("updated_at DESC, id DESC").Limit(limit + 1).Find(&out).Error
	return out, err
}

func (r *AgentRepo) CreateMessage(ctx context.Context, m *model.AgentMessage) error {
	err := r.db.WithContext(ctx).Create(m).Error
	if isDuplicate(err) {
		return repository.ErrDuplicateKey
	}
	return err
}

func (r *AgentRepo) CreatePatrolMessage(ctx context.Context, m *model.AgentMessage, from, to time.Time, maxNonDanger int64) error {
	if m == nil || m.FamilyID == "" || m.Visibility != "family" || m.DedupKey == nil || !from.Before(to) || maxNonDanger < 1 {
		return repository.ErrInvalidQuery
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var family model.Family
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND deleted_at IS NULL", m.FamilyID).First(&family).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return repository.ErrNotFound
			}
			return err
		}
		if m.Severity != "danger" {
			var count int64
			if err := tx.Model(&model.AgentMessage{}).Where("family_id = ? AND visibility = 'family' AND severity <> 'danger' AND generated_at >= ? AND generated_at < ? AND deleted_at IS NULL", m.FamilyID, from, to).Count(&count).Error; err != nil {
				return err
			}
			if count >= maxNonDanger {
				return repository.ErrDailyLimit
			}
		}
		if err := tx.Create(m).Error; err != nil {
			if isDuplicate(err) {
				return repository.ErrDuplicateKey
			}
			return err
		}
		return nil
	})
}

func (r *AgentRepo) FindMessage(ctx context.Context, familyID, id string) (*model.AgentMessage, error) {
	var m model.AgentMessage
	err := r.db.WithContext(ctx).Where("id = ? AND family_id = ? AND deleted_at IS NULL", id, familyID).First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, repository.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *AgentRepo) FindMessageByDedup(ctx context.Context, familyID, dedupKey string) (*model.AgentMessage, error) {
	var m model.AgentMessage
	err := r.db.WithContext(ctx).Where("family_id = ? AND dedup_key = ? AND deleted_at IS NULL", familyID, dedupKey).First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, repository.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *AgentRepo) FindMessageByClientID(ctx context.Context, familyID, userID, sessionID, clientMessageID string) (*model.AgentMessage, error) {
	var m model.AgentMessage
	err := r.db.WithContext(ctx).Where("family_id = ? AND user_id = ? AND session_id = ? AND client_message_id = ? AND role = 'assistant' AND deleted_at IS NULL", familyID, userID, sessionID, clientMessageID).First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, repository.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *AgentRepo) FindUserMessageByClientID(ctx context.Context, familyID, userID, clientMessageID string) (*model.AgentMessage, error) {
	var m model.AgentMessage
	err := r.db.WithContext(ctx).Where("family_id = ? AND user_id = ? AND client_message_id = ? AND role = 'user' AND deleted_at IS NULL", familyID, userID, clientMessageID).
		Order("created_at ASC, id ASC").First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, repository.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *AgentRepo) ListMessages(ctx context.Context, q repository.AgentMessageQuery) ([]*model.AgentMessage, error) {
	limit := q.Limit
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	tx := r.db.WithContext(ctx).Where("family_id = ? AND deleted_at IS NULL", q.FamilyID)
	if q.SessionID != "" {
		tx = tx.Where("session_id = ?", q.SessionID)
	}
	if q.Type != "" {
		tx = tx.Where("type = ?", q.Type)
	}
	if q.Status != "" {
		tx = tx.Where("action_status = ?", q.Status)
	}
	if q.Visibility != "" {
		tx = tx.Where("visibility = ?", q.Visibility)
	}
	if q.ExcludeDismissed {
		tx = tx.Where("(display_status IS NULL OR display_status <> ?) AND (action_status IS NULL OR action_status <> ?)", "dismissed", "dismissed")
	}
	if q.ExcludeTools {
		tx = tx.Where("role <> ?", "tool")
	}
	if t, err := parseAgentCursor(q.Before); err == nil && !t.IsZero() {
		tx = tx.Where("(generated_at, id) < (?, ?)", t, beforeCursorID(q.Before))
	}
	var out []*model.AgentMessage
	err := tx.Order("generated_at DESC, id DESC").Limit(limit + 1).Find(&out).Error
	return out, err
}

func (r *AgentRepo) CountNonDangerMessages(ctx context.Context, familyID string, from, to time.Time) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.AgentMessage{}).
		Where("family_id = ? AND visibility = ? AND severity <> ? AND generated_at >= ? AND generated_at < ? AND deleted_at IS NULL", familyID, "family", "danger", from, to).
		Count(&count).Error
	return count, err
}

func (r *AgentRepo) UpdateMessage(ctx context.Context, familyID string, m *model.AgentMessage) error {
	return r.db.WithContext(ctx).Model(&model.AgentMessage{}).Where("id = ? AND family_id = ? AND deleted_at IS NULL", m.ID, familyID).Updates(map[string]any{
		"draft_payload": m.DraftPayload, "draft_version": m.DraftVersion, "action_status": m.ActionStatus, "display_status": m.DisplayStatus,
		"draft_expires_at": m.DraftExpiresAt, "confirmed_reminder_id": m.ConfirmedReminderID, "updated_at": time.Now().UTC(),
	}).Error
}

func (r *AgentRepo) UpdateMessageIfVersion(ctx context.Context, familyID string, m *model.AgentMessage, expectedVersion int, expectedStatus string) error {
	tx := r.db.WithContext(ctx).Model(&model.AgentMessage{}).Where("id = ? AND family_id = ? AND draft_version = ? AND deleted_at IS NULL", m.ID, familyID, expectedVersion)
	if expectedStatus == "" {
		tx = tx.Where("(action_status IS NULL OR action_status = '')")
	} else {
		tx = tx.Where("action_status = ?", expectedStatus)
	}
	result := tx.Updates(map[string]any{
		"draft_payload": m.DraftPayload, "draft_version": m.DraftVersion, "action_status": m.ActionStatus, "display_status": m.DisplayStatus,
		"draft_expires_at": m.DraftExpiresAt, "confirmed_reminder_id": m.ConfirmedReminderID, "type": m.Type, "updated_at": time.Now().UTC(),
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return repository.ErrConflict
	}
	return nil
}

func (r *AgentRepo) ConfirmMessageAndCreateReminder(ctx context.Context, familyID, actorUserID, messageID string, expectedVersion int, now time.Time, reminder *model.Reminder) (bool, error) {
	if reminder.FamilyID != familyID || actorUserID == "" {
		return false, repository.ErrNotFound
	}
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return false, tx.Error
	}
	rollback := func(err error) (bool, error) {
		_ = tx.Rollback()
		return false, err
	}
	var msg model.AgentMessage
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND family_id = ? AND deleted_at IS NULL", messageID, familyID).First(&msg).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return rollback(repository.ErrNotFound)
		}
		return rollback(err)
	}
	var member model.Member
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("family_id = ? AND user_id = ? AND deleted_at IS NULL", familyID, actorUserID).
		First(&member).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return rollback(repository.ErrNotFound)
		}
		return rollback(err)
	}
	// Private drafts retain the session owner's permissions inside the transaction,
	// including idempotent confirmation of an already confirmed draft.
	if msg.Visibility == "private" {
		var session model.AgentSession
		if msg.Role == "tool" || msg.SessionID == "" {
			return rollback(repository.ErrNotFound)
		}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND family_id = ? AND user_id = ? AND deleted_at IS NULL", msg.SessionID, familyID, actorUserID).First(&session).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return rollback(repository.ErrNotFound)
			}
			return rollback(err)
		}
	} else if msg.Visibility != "family" || msg.Role == "tool" {
		return rollback(repository.ErrNotFound)
	}
	if msg.ActionStatus == "confirmed" && msg.ConfirmedReminderID != "" {
		var existing model.Reminder
		if err := tx.Where("id = ? AND family_id = ? AND deleted_at IS NULL", msg.ConfirmedReminderID, familyID).First(&existing).Error; err != nil {
			return rollback(err)
		}
		if err := tx.Commit().Error; err != nil {
			return false, err
		}
		*reminder = existing
		return false, nil
	}
	if msg.DraftVersion != expectedVersion || msg.ActionStatus != "pending" || msg.DraftExpiresAt == nil || !msg.DraftExpiresAt.After(now) {
		return rollback(repository.ErrConflict)
	}
	if reminder.ScheduledAt == nil || !reminder.ScheduledAt.After(now) {
		return rollback(repository.ErrConflict)
	}
	if reminder.CatID != "both" {
		var cat model.Cat
		if err := tx.Where("id = ? AND family_id = ? AND deleted_at IS NULL", reminder.CatID, familyID).First(&cat).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return rollback(repository.ErrNotFound)
			}
			return rollback(err)
		}
	}
	if err := tx.Create(reminder).Error; err != nil {
		return rollback(err)
	}
	result := tx.Model(&model.AgentMessage{}).Where("id = ? AND family_id = ? AND draft_version = ? AND action_status = 'pending' AND deleted_at IS NULL", messageID, familyID, expectedVersion).Updates(map[string]any{
		"action_status": "confirmed", "confirmed_reminder_id": reminder.ID, "updated_at": time.Now().UTC(),
	})
	if result.Error != nil {
		return rollback(result.Error)
	}
	if result.RowsAffected != 1 {
		return rollback(repository.ErrConflict)
	}
	audit := &model.AuditLog{
		Base:     model.Base{ID: id.ULIDGenerator{}.New(), CreatedBy: actorUserID, CreatedAt: now, UpdatedAt: now},
		FamilyID: familyID, UserID: actorUserID, Action: "agent_reminder_confirm", Resource: messageID,
		Detail: "reminder_id=" + reminder.ID,
	}
	if err := tx.Create(audit).Error; err != nil {
		return rollback(err)
	}
	if err := tx.Commit().Error; err != nil {
		return false, err
	}
	return true, nil
}

func parseAgentCursor(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	parts := splitCursor(s)
	if len(parts) != 2 {
		return time.Time{}, errors.New("invalid cursor")
	}
	return time.Parse(time.RFC3339Nano, parts[0])
}
func beforeCursorID(s string) string {
	p := splitCursor(s)
	if len(p) == 2 {
		return p[1]
	}
	return ""
}
func splitCursor(s string) []string {
	for i, r := range s {
		if r == '|' {
			return []string{s[:i], s[i+1:]}
		}
	}
	return nil
}
