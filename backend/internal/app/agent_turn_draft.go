package app

import (
	"context"
	"crypto/sha256"

	"github.com/meowhome/backend/internal/domain/model"
	"github.com/meowhome/backend/internal/domain/repository"
	apperr "github.com/meowhome/backend/internal/platform/errors"
	"github.com/oklog/ulid/v2"
)

// draftIDForTurn 是当前用户消息对应的唯一草稿目标。重试使用同一个主键。
func draftIDForTurn(turnID string) string {
	hash := sha256.Sum256([]byte("agent-reminder-draft-v1:" + turnID))
	var value ulid.ULID
	copy(value[:], hash[:16])
	return value.String()
}

func (s *AgentService) ensureTurnDraftTarget(ctx context.Context, scope AgentToolScope) (*model.AgentMessage, error) {
	if scope.FamilyID == "" || scope.UserID == "" || scope.SessionID == "" || scope.TurnID == "" {
		return nil, apperr.InvalidRequest(apperr.CodeValidationFailed, "draft turn scope is incomplete")
	}
	if session, err := s.repo.FindSession(ctx, scope.FamilyID, scope.UserID, scope.SessionID); err != nil || session == nil {
		return nil, apperr.NotFound(apperr.CodeNotFound, "agent session not found")
	}
	messageID := draftIDForTurn(scope.TurnID)
	now := s.clock().UTC()
	target := &model.AgentMessage{Base: model.Base{ID: messageID, CreatedBy: scope.UserID, CreatedAt: now, UpdatedAt: now}, FamilyID: scope.FamilyID, SessionID: scope.SessionID, UserID: scope.UserID, TurnID: scope.TurnID, Role: string(RoleAssistant), Visibility: string(VisibilityPrivate), Type: AgentTypeReminderDraft, Severity: "info", Title: "待确认提醒", Body: "请核对提醒内容并手动确认。", GeneratedAt: now, Model: agentChatPromptVersion, Disclaimer: "提醒草稿未经确认不会创建正式提醒。"}
	if err := s.repo.CreateMessage(ctx, target); err == nil {
		return target, nil
	} else if err != repository.ErrDuplicateKey {
		return nil, err
	}
	existing, err := s.repo.FindMessage(ctx, scope.FamilyID, messageID)
	if err != nil {
		return nil, err
	}
	if existing == nil || existing.FamilyID != scope.FamilyID || existing.SessionID != scope.SessionID || existing.UserID != scope.UserID || existing.TurnID != scope.TurnID || existing.Visibility != string(VisibilityPrivate) || existing.Type != AgentTypeReminderDraft {
		return nil, apperr.Conflict(CodeAgentConflict, "draft turn conflict")
	}
	return existing, nil
}
