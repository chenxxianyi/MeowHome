package mysql

import (
	"context"
	"github.com/meowhome/backend/internal/domain/model"
	"github.com/meowhome/backend/internal/domain/repository"
)

func (r *AgentRepo) ClaimEnhancement(ctx context.Context, m *model.AgentMessage) (bool, error) {
	result := r.db.WithContext(ctx).Model(&model.AgentMessage{}).Where("id = ? AND family_id = ? AND visibility = 'family' AND severity <> 'danger' AND draft_version = ? AND COALESCE(action_status,'') = ? AND COALESCE(display_status,'') = ? AND COALESCE(enhance_status,'') = '' AND deleted_at IS NULL", m.ID, m.FamilyID, m.DraftVersion, m.ActionStatus, m.DisplayStatus).Update("enhance_status", "running")
	return result.RowsAffected == 1, result.Error
}

func (r *AgentRepo) SaveEnhancement(ctx context.Context, m *model.AgentMessage) error {
	result := r.db.WithContext(ctx).Model(&model.AgentMessage{}).Where("id = ? AND family_id = ? AND draft_version = ? AND COALESCE(action_status,'') = ? AND COALESCE(display_status,'') = ? AND enhance_status = 'running' AND deleted_at IS NULL", m.ID, m.FamilyID, m.DraftVersion, m.ActionStatus, m.DisplayStatus).Updates(map[string]any{"enhanced_body": m.EnhancedBody, "enhance_status": m.EnhanceStatus, "enhance_model": m.EnhanceModel, "enhance_prompt_version": m.EnhancePromptVersion, "prompt_tokens": m.PromptTokens, "completion_tokens": m.CompletionTokens, "total_tokens": m.TotalTokens, "duration_ms": m.DurationMS})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return repository.ErrConflict
	}
	return nil
}
