package mysql

import (
	"context"
	"errors"
	"time"

	"github.com/meowhome/backend/internal/domain/model"
	"github.com/meowhome/backend/internal/domain/repository"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (r *AgentRepo) ClaimChatTurn(ctx context.Context, session *model.AgentSession, user *model.AgentMessage, requestedSessionID, token string, now, leaseUntil time.Time) (bool, error) {
	acquired := false
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// A brief family row lock also protects first requests without a session ID.
		var family model.Family
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND deleted_at IS NULL", user.FamilyID).First(&family).Error; err != nil {
			return agentNotFound(err)
		}
		var member model.Member
		if err := tx.Where("family_id = ? AND user_id = ? AND deleted_at IS NULL", user.FamilyID, user.UserID).First(&member).Error; err != nil {
			return agentNotFound(err)
		}
		var existing model.AgentMessage
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("family_id = ? AND user_id = ? AND client_message_id = ? AND role = 'user' AND deleted_at IS NULL", user.FamilyID, user.UserID, *user.ClientMessageID).First(&existing).Error
		found := err == nil
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if found {
			if existing.Body != user.Body || (requestedSessionID != "" && existing.SessionID != requestedSessionID) {
				return repository.ErrConflict
			}
			var storedSession model.AgentSession
			if err := tx.Where("id = ? AND family_id = ? AND user_id = ? AND deleted_at IS NULL", existing.SessionID, user.FamilyID, user.UserID).First(&storedSession).Error; err != nil {
				return agentNotFound(err)
			}
			*session = storedSession
			*user = existing
			var count int64
			if err := tx.Model(&model.AgentMessage{}).Where("family_id = ? AND user_id = ? AND client_message_id = ? AND role = 'assistant' AND deleted_at IS NULL", user.FamilyID, user.UserID, *user.ClientMessageID).Count(&count).Error; err != nil {
				return err
			}
			if count > 0 || (user.RunStatus == "running" && user.RunLeaseUntil != nil && user.RunLeaseUntil.After(now)) {
				return nil
			}
		} else if requestedSessionID != "" {
			var storedSession model.AgentSession
			if err := tx.Where("id = ? AND family_id = ? AND user_id = ? AND deleted_at IS NULL", requestedSessionID, user.FamilyID, user.UserID).First(&storedSession).Error; err != nil {
				return agentNotFound(err)
			}
			*session = storedSession
		}
		var busy int64
		if err := tx.Model(&model.AgentMessage{}).Where("family_id = ? AND session_id = ? AND id <> ? AND role = 'user' AND run_status = 'running' AND run_lease_until > ? AND deleted_at IS NULL", user.FamilyID, session.ID, user.ID, now).Count(&busy).Error; err != nil {
			return err
		}
		if busy > 0 {
			return repository.ErrConflict
		}
		if !found && requestedSessionID == "" {
			if err := tx.Create(session).Error; err != nil {
				return err
			}
		}
		user.SessionID, user.TurnID = session.ID, user.ID
		user.RunStatus, user.RunToken, user.RunLeaseUntil = "running", token, &leaseUntil
		if !found {
			if err := tx.Create(user).Error; err != nil {
				return err
			}
		} else {
			if err := tx.Model(&model.AgentMessage{}).Where("id = ? AND family_id = ?", user.ID, user.FamilyID).Updates(map[string]any{"run_status": "running", "run_token": token, "run_lease_until": leaseUntil, "turn_id": user.ID, "run_error": ""}).Error; err != nil {
				return err
			}
		}
		acquired = true
		return nil
	})
	return acquired, err
}

func (r *AgentRepo) FinishChatTurn(ctx context.Context, user *model.AgentMessage, token string, answer *model.AgentMessage) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var current model.AgentMessage
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND family_id = ? AND user_id = ? AND deleted_at IS NULL", user.ID, user.FamilyID, user.UserID).First(&current).Error; err != nil {
			return agentNotFound(err)
		}
		if current.RunToken != token || current.RunStatus != "running" {
			return repository.ErrConflict
		}
		if answer != nil {
			if answer.TurnID != current.ID || answer.SessionID != current.SessionID || answer.UserID != current.UserID || answer.FamilyID != current.FamilyID || answer.Visibility != "private" {
				return repository.ErrConflict
			}
			if err := tx.Create(answer).Error; err != nil {
				return err
			}
			if err := tx.Model(&model.AgentSession{}).Where("id = ? AND family_id = ? AND user_id = ? AND deleted_at IS NULL", current.SessionID, current.FamilyID, current.UserID).Updates(map[string]any{"last_message_at": answer.GeneratedAt, "updated_at": answer.GeneratedAt}).Error; err != nil {
				return err
			}
		}
		return tx.Model(&model.AgentMessage{}).Where("id = ? AND family_id = ?", user.ID, user.FamilyID).Updates(map[string]any{"run_status": user.RunStatus, "run_token": "", "run_lease_until": nil, "run_error": user.RunError, "duration_ms": user.DurationMS}).Error
	})
}

func agentNotFound(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return repository.ErrNotFound
	}
	return err
}
