// Package repository 定义仓储接口。
package repository

import (
	"context"
	"errors"
	"time"

	"github.com/meowhome/backend/internal/domain/model"
)

// ErrDuplicateKey 唯一键冲突。
var ErrDuplicateKey = errors.New("duplicate key")

// ErrNotFound 资源不存在。
var ErrNotFound = errors.New("not found")

// ErrConflict 表示乐观锁或状态机冲突。
var ErrConflict = errors.New("conflict")

// UserRepo 用户仓储。
type UserRepo interface {
	Create(ctx context.Context, u *model.User) error
	FindByID(ctx context.Context, id string) (*model.User, error)
	FindByEmail(ctx context.Context, email string) (*model.User, error)
}

// FamilyRepo 家庭仓储。
type FamilyRepo interface {
	Create(ctx context.Context, f *model.Family) error
	FindByID(ctx context.Context, id string) (*model.Family, error)
	List(ctx context.Context, beforeID string, limit int) ([]*model.Family, error)
	Update(ctx context.Context, f *model.Family) error
	Delete(ctx context.Context, id string) error
}

// MemberRepo 成员仓储。
type MemberRepo interface {
	Create(ctx context.Context, m *model.Member) error
	FindByFamily(ctx context.Context, familyID string) ([]*model.Member, error)
	FindByUser(ctx context.Context, userID string) ([]*model.Member, error)
	UpdateRole(ctx context.Context, memberID, role string) error
	Delete(ctx context.Context, memberID string) error
}

// CatRepo 猫咪仓储。
type CatRepo interface {
	Create(ctx context.Context, c *model.Cat) error
	FindByID(ctx context.Context, id string) (*model.Cat, error)
	ListByFamily(ctx context.Context, familyID string) ([]*model.Cat, error)
	Update(ctx context.Context, c *model.Cat) error
	Delete(ctx context.Context, id string) error
}

// CatHealthProfileRepo 健康档案仓储。
type CatHealthProfileRepo interface {
	Create(ctx context.Context, p *model.CatHealthProfile) error
	FindByCatID(ctx context.Context, catID string) (*model.CatHealthProfile, error)
	Update(ctx context.Context, p *model.CatHealthProfile) error
}

// RefreshTokenRepo 刷新令牌仓储。
type RefreshTokenRepo interface {
	Create(ctx context.Context, t *model.RefreshToken) error
	FindByHash(ctx context.Context, hash string) (*model.RefreshToken, error)
	Revoke(ctx context.Context, id string) error
	RevokeAllForUser(ctx context.Context, userID string) error
}

// DailyRecordRepo 日常记录仓储。
// 注意：model.Record 与 model.DailyRecord 都映射到 daily_records 表，
// 这里统一使用 DailyRecord（支持多猫与就诊关联），避免两个模型同名冲突。
type DailyRecordRepo interface {
	Create(ctx context.Context, r *model.DailyRecord) error
	FindByID(ctx context.Context, id string) (*model.DailyRecord, error)
	ListByFamily(ctx context.Context, q DailyRecordQuery) ([]*model.DailyRecord, error)
	Delete(ctx context.Context, id string) error
}

// DailyRecordQuery 记录查询条件。
type DailyRecordQuery struct {
	FamilyID         string
	CatID            string   // 命中 cat_ids JSON 数组中的任一元素
	Types            []string // 空表示不限
	From             string   // RFC3339 或 YYYY-MM-DD
	To               string
	BeforeOccurredAt *time.Time // 与 BeforeID 组成稳定倒序游标
	BeforeID         string
	Limit            int
}

// TimelineRepo 时光事件仓储。
type TimelineRepo interface {
	Create(ctx context.Context, e *model.TimelineEvent) error
	ListByFamily(ctx context.Context, familyID string, limit int) ([]*model.TimelineEvent, error)
}

// AnalysisReportRepo AI 分析报告仓储。
type AnalysisReportRepo interface {
	Create(ctx context.Context, r *model.AnalysisReport) error
	LatestByFamily(ctx context.Context, familyID string) (*model.AnalysisReport, error)
}

// HealthRepo 健康聚合仓储。
type HealthRepo interface {
	Aggregation(ctx context.Context, q AggregationQuery) (*model.Aggregation, error)
}

// AggregationQuery 聚合查询条件。
type AggregationQuery struct {
	FamilyID string
	CatID    string
	From     string
	To       string
}

// ReminderRepo 提醒仓储。
type ReminderRepo interface {
	Create(ctx context.Context, r *model.Reminder) error
	FindByID(ctx context.Context, id string) (*model.Reminder, error)
	List(ctx context.Context, q ReminderQuery) ([]*model.Reminder, error)
	Update(ctx context.Context, r *model.Reminder) error
	Delete(ctx context.Context, id string) error
}

// ReminderQuery 提醒查询条件。
type ReminderQuery struct {
	FamilyID string
	CatID    string
	Status   string
}

// AgentMessageQuery Agent 消息查询条件。
type AgentMessageQuery struct {
	FamilyID         string
	SessionID        string
	Type             string
	Status           string
	Visibility       string
	ExcludeDismissed bool
	Before           string
	Limit            int
}

// AgentRepo 持久化 Agent 会话、消息和提醒草稿。
type AgentRepo interface {
	CreateSession(ctx context.Context, s *model.AgentSession) error
	UpdateSession(ctx context.Context, s *model.AgentSession) error
	FindSession(ctx context.Context, id string) (*model.AgentSession, error)
	ListSessions(ctx context.Context, familyID, userID, before string, limit int) ([]*model.AgentSession, error)
	CreateMessage(ctx context.Context, m *model.AgentMessage) error
	FindMessage(ctx context.Context, id string) (*model.AgentMessage, error)
	FindMessageByDedup(ctx context.Context, familyID, dedupKey string) (*model.AgentMessage, error)
	FindMessageByClientID(ctx context.Context, sessionID, clientMessageID string) (*model.AgentMessage, error)
	FindAnyMessageByClientID(ctx context.Context, familyID, userID, clientMessageID string) (*model.AgentMessage, error)
	ListMessages(ctx context.Context, q AgentMessageQuery) ([]*model.AgentMessage, error)
	UpdateMessage(ctx context.Context, m *model.AgentMessage) error
	UpdateMessageIfVersion(ctx context.Context, m *model.AgentMessage, expectedVersion int) error
	// ConfirmMessageAndCreateReminder 在同一事务中锁定消息、创建提醒并确认草稿。
	// created=false 表示并发请求已经完成确认，返回已有提醒。
	ConfirmMessageAndCreateReminder(ctx context.Context, messageID string, expectedVersion int, now time.Time, reminder *model.Reminder) (created bool, err error)
}

// InventoryRepo 库存仓储。
type InventoryRepo interface {
	Create(ctx context.Context, i *model.InventoryItem) error
	FindByID(ctx context.Context, id string) (*model.InventoryItem, error)
	ListByFamily(ctx context.Context, familyID string) ([]*model.InventoryItem, error)
	Update(ctx context.Context, i *model.InventoryItem) error
	Delete(ctx context.Context, id string) error
}

// ExpenseRepo 支出仓储。
type ExpenseRepo interface {
	Create(ctx context.Context, e *model.Expense) error
	FindByID(ctx context.Context, id string) (*model.Expense, error)
	List(ctx context.Context, q ExpenseQuery) ([]*model.Expense, error)
}

// ExpenseQuery 支出查询条件。
type ExpenseQuery struct {
	FamilyID string
	Month    string
}

// MediaRepo 媒体仓储。
type MediaRepo interface {
	Create(ctx context.Context, m *model.Media) error
	FindByID(ctx context.Context, id string) (*model.Media, error)
	Update(ctx context.Context, m *model.Media) error
	Delete(ctx context.Context, id string) error
}

// AuditRepo 审计日志仓储。
type AuditRepo interface {
	Create(ctx context.Context, a *model.AuditLog) error
}

// AIRepo AI 会话仓储。
type AIRepo interface {
	Create(ctx context.Context, s *model.AIParseSession) error
	FindByID(ctx context.Context, id string) (*model.AIParseSession, error)
	Update(ctx context.Context, s *model.AIParseSession) error
	Delete(ctx context.Context, id string) error
}

// ExportRepo 导出仓储。
type ExportRepo interface {
	Create(ctx context.Context, e *model.ExportJob) error
	FindByID(ctx context.Context, id string) (*model.ExportJob, error)
	Update(ctx context.Context, e *model.ExportJob) error
}
