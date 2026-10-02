package model

import "time"

// AgentSession 是用户在单个家庭内的私有对话会话。
type AgentSession struct {
	Base
	FamilyID      string     `gorm:"index;type:varchar(26);not null"`
	UserID        string     `gorm:"index;type:varchar(26);not null"`
	Title         string     `gorm:"type:varchar(200)"`
	Status        string     `gorm:"type:varchar(20);not null;default:active"`
	LastMessageAt *time.Time `gorm:"index"`
}

func (AgentSession) TableName() string { return "ai_agent_sessions" }

// AgentMessage 同时承载巡检消息、对话消息和待确认草稿。
// JSON 字段使用 TEXT 保存，避免把未校验的模型原始响应直接暴露给客户端。
type AgentMessage struct {
	Base
	FamilyID            string     `gorm:"index;type:varchar(26);not null"`
	SessionID           string     `gorm:"index;type:varchar(26)"`
	UserID              string     `gorm:"index;type:varchar(26)"`
	Role                string     `gorm:"type:varchar(16);not null"`
	Visibility          string     `gorm:"index;type:varchar(16);not null"`
	CatID               string     `gorm:"index;type:varchar(26)"`
	Type                string     `gorm:"index;type:varchar(32);not null"`
	Severity            string     `gorm:"type:varchar(16);not null;default:info"`
	Title               string     `gorm:"type:varchar(200);not null"`
	Body                string     `gorm:"type:text;not null"`
	Evidence            string     `gorm:"type:text"`
	ActionSuggestions   string     `gorm:"type:text"`
	DraftPayload        string     `gorm:"type:text"`
	DraftVersion        int        `gorm:"not null;default:0"`
	ActionStatus        string     `gorm:"index;type:varchar(16)"`
	DisplayStatus       string     `gorm:"type:varchar(16)"`
	GeneratedAt         time.Time  `gorm:"index;not null"`
	DraftExpiresAt      *time.Time `gorm:"index"`
	ConfirmedReminderID string     `gorm:"type:varchar(26)"`
	Model               string     `gorm:"type:varchar(80);not null"`
	Disclaimer          string     `gorm:"type:text"`
	// DedupKey 仅用于巡检消息；聊天消息必须保持 NULL，不能用空字符串占用唯一键。
	DedupKey *string `gorm:"uniqueIndex;type:varchar(255)"`
	// nil 表示非聊天消息。MySQL 唯一索引允许多个 NULL；空字符串会让巡检消息互相冲突。
	ClientMessageID *string `gorm:"type:varchar(128)"`
	// ToolCallID 关联一次模型工具调用；工具结果仅存受控摘要，不向客户端透出原始响应。
	ToolCallID  string     `gorm:"type:varchar(128)"`
	ToolName    string     `gorm:"type:varchar(64)"`
	TurnID      string     `gorm:"type:varchar(26)"`
	RuleID      string     `gorm:"type:varchar(16)"`
	RuleVersion string     `gorm:"type:varchar(32)"`
	Scope       string     `gorm:"type:varchar(255)"`
	WindowStart *time.Time `gorm:"index"`
	WindowEnd   *time.Time `gorm:"index"`
}

func (AgentMessage) TableName() string { return "ai_agent_messages" }
