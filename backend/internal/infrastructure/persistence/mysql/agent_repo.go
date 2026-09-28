package mysql

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/meowhome/backend/internal/domain/model"
	"github.com/meowhome/backend/internal/domain/repository"
)

type AgentRepo struct{ db *gorm.DB }

func NewAgentRepo(db *gorm.DB) *AgentRepo { return &AgentRepo{db: db} }

var _ repository.AgentRepo = (*AgentRepo)(nil)

func (r *AgentRepo) CreateSession(ctx context.Context, s *model.AgentSession) error {
	return r.db.WithContext(ctx).Create(s).Error
}

func (r *AgentRepo) UpdateSession(ctx context.Context, s *model.AgentSession) error {
	return r.db.WithContext(ctx).Save(s).Error
}

func (r *AgentRepo) FindSession(ctx context.Context, id string) (*model.AgentSession, error) {
	var s model.AgentSession
	err := r.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&s).Error
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
		tx = tx.Where("(last_message_at, id) < (?, ?)", t, beforeCursorID(before))
	}
	var out []*model.AgentSession
	err := tx.Order("last_message_at DESC, id DESC").Limit(limit + 1).Find(&out).Error
	return out, err
}

func (r *AgentRepo) CreateMessage(ctx context.Context, m *model.AgentMessage) error {
	err := r.db.WithContext(ctx).Create(m).Error
	if isDuplicate(err) {
		return repository.ErrDuplicateKey
	}
	return err
}

func (r *AgentRepo) FindMessage(ctx context.Context, id string) (*model.AgentMessage, error) {
	var m model.AgentMessage
	err := r.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&m).Error
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

func (r *AgentRepo) FindMessageByClientID(ctx context.Context, sessionID, clientMessageID string) (*model.AgentMessage, error) {
	var m model.AgentMessage
	err := r.db.WithContext(ctx).Where("session_id = ? AND client_message_id = ? AND role = 'assistant' AND deleted_at IS NULL", sessionID, clientMessageID).First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, repository.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *AgentRepo) FindAnyMessageByClientID(ctx context.Context, familyID, userID, clientMessageID string) (*model.AgentMessage, error) {
	var m model.AgentMessage
	err := r.db.WithContext(ctx).Where("family_id = ? AND user_id = ? AND client_message_id = ? AND deleted_at IS NULL", familyID, userID, clientMessageID).
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
		tx = tx.Where("action_status IS NULL OR action_status <> ?", "dismissed")
	}
	if t, err := parseAgentCursor(q.Before); err == nil && !t.IsZero() {
		tx = tx.Where("(generated_at, id) < (?, ?)", t, beforeCursorID(q.Before))
	}
	var out []*model.AgentMessage
	err := tx.Order("generated_at DESC, id DESC").Limit(limit + 1).Find(&out).Error
	return out, err
}

func (r *AgentRepo) UpdateMessage(ctx context.Context, m *model.AgentMessage) error {
	return r.db.WithContext(ctx).Model(&model.AgentMessage{}).Where("id = ? AND deleted_at IS NULL", m.ID).Updates(map[string]any{
		"draft_payload": m.DraftPayload, "draft_version": m.DraftVersion, "action_status": m.ActionStatus,
		"draft_expires_at": m.DraftExpiresAt, "confirmed_reminder_id": m.ConfirmedReminderID, "updated_at": time.Now().UTC(),
	}).Error
}

func (r *AgentRepo) UpdateMessageIfVersion(ctx context.Context, m *model.AgentMessage, expectedVersion int) error {
	result := r.db.WithContext(ctx).Model(&model.AgentMessage{}).Where("id = ? AND draft_version = ? AND deleted_at IS NULL", m.ID, expectedVersion).Updates(map[string]any{
		"draft_payload": m.DraftPayload, "draft_version": m.DraftVersion, "action_status": m.ActionStatus,
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

func (r *AgentRepo) ConfirmMessageAndCreateReminder(ctx context.Context, messageID string, expectedVersion int, now time.Time, reminder *model.Reminder) (bool, error) {
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return false, tx.Error
	}
	rollback := func(err error) (bool, error) {
		_ = tx.Rollback()
		return false, err
	}
	var msg model.AgentMessage
	if err := tx.Set("gorm:query_option", "FOR UPDATE").Where("id = ? AND deleted_at IS NULL", messageID).First(&msg).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return rollback(repository.ErrNotFound)
		}
		return rollback(err)
	}
	if msg.ActionStatus == "confirmed" && msg.ConfirmedReminderID != "" {
		var existing model.Reminder
		if err := tx.Where("id = ? AND deleted_at IS NULL", msg.ConfirmedReminderID).First(&existing).Error; err != nil {
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
	if err := tx.Create(reminder).Error; err != nil {
		return rollback(err)
	}
	result := tx.Model(&model.AgentMessage{}).Where("id = ? AND draft_version = ? AND action_status = 'pending' AND deleted_at IS NULL", messageID, expectedVersion).Updates(map[string]any{
		"action_status": "confirmed", "confirmed_reminder_id": reminder.ID, "updated_at": time.Now().UTC(),
	})
	if result.Error != nil {
		return rollback(result.Error)
	}
	if result.RowsAffected != 1 {
		return rollback(repository.ErrConflict)
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
