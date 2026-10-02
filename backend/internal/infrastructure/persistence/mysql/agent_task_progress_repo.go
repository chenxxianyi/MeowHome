package mysql

import (
	"context"
	"errors"
	"time"

	"github.com/meowhome/backend/internal/domain/repository"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type agentTaskProgressRow struct {
	FamilyID        string `gorm:"primaryKey;type:varchar(26)"`
	TaskKey         string `gorm:"primaryKey;type:varchar(64)"`
	Status          string `gorm:"type:varchar(16);not null"`
	LastSuccessAt   *time.Time
	LastFailureAt   *time.Time
	CursorCreatedAt *time.Time
	CursorID        string `gorm:"type:varchar(26)"`
}

func (agentTaskProgressRow) TableName() string { return "ai_agent_task_progress" }

type AgentTaskProgressRepo struct{ db *gorm.DB }

func NewAgentTaskProgressRepo(db *gorm.DB) *AgentTaskProgressRepo {
	return &AgentTaskProgressRepo{db: db}
}

var _ repository.AgentTaskProgressRepo = (*AgentTaskProgressRepo)(nil)

func (r *AgentTaskProgressRepo) Get(ctx context.Context, familyID, taskKey string) (*repository.AgentTaskProgress, error) {
	if familyID == "" || taskKey == "" {
		return nil, repository.ErrInvalidQuery
	}
	var row agentTaskProgressRow
	err := r.db.WithContext(ctx).Where("family_id = ? AND task_key = ?", familyID, taskKey).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, repository.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &repository.AgentTaskProgress{FamilyID: row.FamilyID, TaskKey: row.TaskKey, Status: row.Status, LastSuccessAt: row.LastSuccessAt, LastFailureAt: row.LastFailureAt, CursorCreatedAt: row.CursorCreatedAt, CursorID: row.CursorID}, nil
}

func (r *AgentTaskProgressRepo) Save(ctx context.Context, p *repository.AgentTaskProgress) error {
	if p == nil || p.FamilyID == "" || p.TaskKey == "" {
		return repository.ErrInvalidQuery
	}
	row := agentTaskProgressRow{FamilyID: p.FamilyID, TaskKey: p.TaskKey, Status: p.Status, LastSuccessAt: p.LastSuccessAt, LastFailureAt: p.LastFailureAt, CursorCreatedAt: p.CursorCreatedAt, CursorID: p.CursorID}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "family_id"}, {Name: "task_key"}}, DoUpdates: clause.AssignmentColumns([]string{"status", "last_success_at", "last_failure_at", "cursor_created_at", "cursor_id"})}).Create(&row).Error
}
