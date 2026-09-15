// Package mysql 实现 AI 会话与分析报告仓储。
package mysql

import (
	"context"

	"gorm.io/gorm"

	"github.com/meowhome/backend/internal/domain/model"
	"github.com/meowhome/backend/internal/domain/repository"
)

// AIRepo 实现 repository.AIRepo。
type AIRepo struct {
	db *gorm.DB
}

// NewAIRepo 创建 AI 仓储。
func NewAIRepo(db *gorm.DB) *AIRepo {
	return &AIRepo{db: db}
}

var _ repository.AIRepo = (*AIRepo)(nil)

// Create 创建解析会话。
func (r *AIRepo) Create(ctx context.Context, s *model.AIParseSession) error {
	return r.db.WithContext(ctx).Create(s).Error
}

// FindByID 按 ID 查找解析会话。
func (r *AIRepo) FindByID(ctx context.Context, id string) (*model.AIParseSession, error) {
	var s model.AIParseSession
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&s).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}
	return &s, nil
}

// Update 更新解析会话。
func (r *AIRepo) Update(ctx context.Context, s *model.AIParseSession) error {
	return r.db.WithContext(ctx).Save(s).Error
}

// Delete 删除解析会话。
func (r *AIRepo) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&model.AIParseSession{}).Error
}

// AnalysisReportRepo 实现 repository.AnalysisReportRepo。
type AnalysisReportRepo struct {
	db *gorm.DB
}

// NewAnalysisReportRepo 创建分析报告仓储。
func NewAnalysisReportRepo(db *gorm.DB) *AnalysisReportRepo {
	return &AnalysisReportRepo{db: db}
}

var _ repository.AnalysisReportRepo = (*AnalysisReportRepo)(nil)

// Create 创建分析报告。
func (r *AnalysisReportRepo) Create(ctx context.Context, rep *model.AnalysisReport) error {
	return r.db.WithContext(ctx).Create(rep).Error
}

// LatestByFamily 取家庭最近一份分析报告。
func (r *AnalysisReportRepo) LatestByFamily(ctx context.Context, familyID string) (*model.AnalysisReport, error) {
	var rep model.AnalysisReport
	if err := r.db.WithContext(ctx).Where("family_id = ?", familyID).
		Order("created_at DESC").First(&rep).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}
	return &rep, nil
}
