package model

import "time"

// RefreshToken 刷新令牌。
type RefreshToken struct {
	Base
	UserID    string    `gorm:"index;type:varchar(26);not null"`
	TokenHash string    `gorm:"type:varchar(255);not null"`
	IssuedAt  time.Time `gorm:"not null"`
	ExpiresAt time.Time `gorm:"not null"`
	RevokedAt *time.Time
}

func (RefreshToken) TableName() string { return "refresh_tokens" }

// Reminder 提醒规则。
type Reminder struct {
	Base
	FamilyID  string `gorm:"index;type:varchar(26);not null"`
	CatID     string `gorm:"type:varchar(64)"` // 猫咪 ID 或 "both"
	Type      string `gorm:"type:varchar(32);not null"`
	Title     string `gorm:"type:varchar(255)"`
	Subtitle  string `gorm:"type:varchar(255)"`
	TimeLabel string `gorm:"column:time_label;type:varchar(64)"`
	State     string `gorm:"type:varchar(20);default:todo"`
	Icon      string `gorm:"type:varchar(64)"`
	Rule      string `gorm:"type:text"` // cron 或自定义规则 JSON
}

func (Reminder) TableName() string { return "reminders" }

// ReminderOccurrence 提醒实例。
type ReminderOccurrence struct {
	Base
	FamilyID    string `gorm:"index;type:varchar(26);not null"`
	ReminderID  string `gorm:"index;type:varchar(26);not null"`
	ScheduledAt string `gorm:"index;not null"`
	Status      string `gorm:"type:varchar(20);default:todo"`
}

func (ReminderOccurrence) TableName() string { return "reminder_occurrences" }

// CareTask 照护任务。
type CareTask struct {
	Base
	FamilyID string `gorm:"index;type:varchar(26);not null"`
	Title    string `gorm:"type:varchar(255);not null"`
	Status   string `gorm:"type:varchar(20);default:todo"`
}

func (CareTask) TableName() string { return "care_tasks" }

// InventoryItem 库存项。
type InventoryItem struct {
	Base
	FamilyID          string `gorm:"index;type:varchar(26);not null"`
	Name              string `gorm:"type:varchar(255);not null"`
	Category          string `gorm:"type:varchar(64)"`
	Unit              string `gorm:"type:varchar(32)"`
	Quantity          int    `gorm:"default:0"`
	LowStockThreshold int    `gorm:"default:0"`
	Expiry            string `gorm:"type:date"`
}

func (InventoryItem) TableName() string { return "inventory_items" }

// InventoryTransaction 库存流水。
type InventoryTransaction struct {
	Base
	FamilyID string `gorm:"index;type:varchar(26);not null"`
	ItemID   string `gorm:"index;type:varchar(26);not null"`
	Change   int    `gorm:"not null"` // 正入负出
	Reason   string `gorm:"type:varchar(120)"`
}

func (InventoryTransaction) TableName() string { return "inventory_transactions" }

// Expense 支出。
type Expense struct {
	Base
	FamilyID   string   `gorm:"index;type:varchar(26);not null"`
	OccurredOn string   `gorm:"type:date"`
	Amount     int64    `gorm:"not null"` // 最小货币单位
	Category   string   `gorm:"type:varchar(64)"`
	Label      string   `gorm:"type:varchar(255)"`
	CatIDs     []string `gorm:"serializer:json;type:text"`
}

func (Expense) TableName() string { return "expenses" }

// Media 媒体资产。
type Media struct {
	Base
	FamilyID         string `gorm:"index;type:varchar(26);not null"`
	ObjectKey        string `gorm:"uniqueIndex;type:varchar(512)"`
	OriginalName     string `gorm:"type:varchar(512)"`
	MIME             string `gorm:"type:varchar(128)"`
	Size             int64
	ProcessingStatus string `gorm:"type:varchar(20);default:uploaded"`
}

func (Media) TableName() string { return "media_assets" }

// TimelineEvent 时光事件。
type TimelineEvent struct {
	Base
	FamilyID   string   `gorm:"index;type:varchar(26);not null"`
	CatIDs     []string `gorm:"serializer:json;type:text"`
	Title      string   `gorm:"type:varchar(255)"`
	Body       string   `gorm:"type:text"`
	MediaID    string   `gorm:"type:varchar(26)"`
	EventType  string   `gorm:"type:varchar(32);default:photo"`
	OccurredOn string   `gorm:"type:date"`
	ImageCount int      `gorm:"default:0"`
}

func (TimelineEvent) TableName() string { return "timeline_events" }

// CatInteraction 猫咪互动。
type CatInteraction struct {
	Base
	FamilyID   string   `gorm:"index;type:varchar(26);not null"`
	CatIDs     []string `gorm:"serializer:json;type:text"`
	Type       string   `gorm:"type:varchar(32)"`
	Notes      string   `gorm:"type:text"`
	OccurredOn string   `gorm:"type:date"`
}

func (CatInteraction) TableName() string { return "cat_interactions" }

// AIParseSession AI 解析会话。
type AIParseSession struct {
	Base
	FamilyID      string `gorm:"index;type:varchar(26);not null"`
	OriginalInput string `gorm:"type:text"`
	Status        string `gorm:"type:varchar(20);default:processing"`
	Model         string `gorm:"type:varchar(120)"`
	Result        string `gorm:"type:longtext"` // 解析出的记录 JSON
}

func (AIParseSession) TableName() string { return "ai_parse_sessions" }

// AnalysisReport AI 分析报告。
type AnalysisReport struct {
	Base
	FamilyID    string `gorm:"index;type:varchar(26);not null"`
	CatID       string `gorm:"type:varchar(26)"`
	RecordCount int    `gorm:"default:0"`
	Content     string `gorm:"type:text"`
	Evidence    string `gorm:"type:text"` // 证据列表 JSON
}

func (AnalysisReport) TableName() string { return "ai_analysis_reports" }

// EvidenceLink AI 证据链接。
type EvidenceLink struct {
	Base
	ReportID     string `gorm:"index;type:varchar(26)"`
	EvidenceType string `gorm:"type:varchar(32)"`
	EvidenceID   string `gorm:"type:varchar(26)"`
}

func (EvidenceLink) TableName() string { return "ai_evidence_links" }

// AuditLog 审计日志。
type AuditLog struct {
	Base
	FamilyID string `gorm:"type:varchar(26)"`
	UserID   string `gorm:"type:varchar(26)"`
	Action   string `gorm:"type:varchar(64);not null"`
	Resource string `gorm:"type:varchar(128)"`
	Detail   string `gorm:"type:text"`
}

func (AuditLog) TableName() string { return "audit_logs" }

// ExportJob 导出任务。
type ExportJob struct {
	Base
	FamilyID string `gorm:"index;type:varchar(26);not null"`
	Status   string `gorm:"type:varchar(20);default:pending"`
	Format   string `gorm:"type:varchar(20)"`
	FileKey  string `gorm:"type:varchar(512)"`
}

func (ExportJob) TableName() string { return "export_jobs" }

// IdempotencyKey 幂等键。
type IdempotencyKey struct {
	Base
	FamilyID string `gorm:"index;type:varchar(26)"`
	Key      string `gorm:"uniqueIndex;type:varchar(255)"`
	Result   string `gorm:"type:text"`
}

func (IdempotencyKey) TableName() string { return "idempotency_keys" }

// Aggregation 健康聚合结果（查询对象）。
type Aggregation struct {
	FamilyID string
	CatID    string
	From     string
	To       string
	Content  string // JSON 结果
}
