// Package app 提供 Agent（猫管家）契约层 DTO。
//
// 本文件是 AG-01「冻结开发契约」的 Go 侧权威定义，字段与 OpenAPI schema、
// 小程序 src/types/agent.ts 一一对应。业务实现（agent_service.go 等）
// 必须从这里取类型，不得另起炉灶。
//
// 设计依据：MeowHome-Agent设计方案.md §3–§5；
// 歧义裁定：MeowHome-Agent开发任务.md 第 2 节（D02/D03/D05/D10/D12/D13）。
package app

import (
	"time"

	"github.com/meowhome/backend/internal/platform/errors"
)

// ── 消息角色 ──────────────────────────────────────────────

// Role 对话消息角色。家庭巡检消息 role=assistant（系统生成）。
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool" // 工具调用结果（仅会话内部可见，不直接对外）
)

// 消息可见性：family=家庭巡检消息，本家庭现有成员可见；
// private=用户对话消息，仅会话创建者可读写（§2.1）。
type Visibility string

const (
	VisibilityFamily  Visibility = "family"
	VisibilityPrivate Visibility = "private"
)

// AgentMessage 一条 Agent 消息（巡检结果或对话回复）。
//
// JSON 字段名与小程序 agent.ts 严格一致；列表接口返回
// {messages, next_cursor}，聊天接口返回 {session_id, message, degraded}。
type AgentMessage struct {
	ID              string              `json:"id"`
	SessionID       string              `json:"session_id,omitempty"` // 对话消息所属会话；巡检消息可空
	Role            Role                `json:"role"`
	Visibility      Visibility          `json:"visibility"`
	FamilyID        string              `json:"family_id,omitempty"` // 会话恢复/审计用；列表接口可省
	UserID          string              `json:"user_id,omitempty"`   // 系统消息可空；私有消息必填
	CatID           string              `json:"cat_id,omitempty"`    // 单猫消息可见的明确归属
	Type            string              `json:"type"`                // 见 AgentMessageType 常量
	Severity        string              `json:"severity"`            // info | warning | danger
	Title           string              `json:"title"`
	Body            string              `json:"body"`
	Evidence        []AgentEvidence     `json:"evidence,omitempty"`
	Actions         []AgentAction       `json:"action_suggestions,omitempty"`
	DraftVersion    int                 `json:"draft_version,omitempty"`  // 消息携带的提醒草稿版本
	DraftReminder   *AgentReminderInput `json:"draft_reminder,omitempty"` // 服务端保存的编辑值
	DraftExpiresAt  *time.Time          `json:"draft_expires_at,omitempty"`
	ActionStatus    string              `json:"action_status,omitempty"`  // pending | confirmed | dismissed | expired
	DisplayStatus   string              `json:"display_status,omitempty"` // dismissed 表示在家庭消息列表隐藏，不改变已确认提醒
	GeneratedAt     time.Time           `json:"generated_at"`
	Model           string              `json:"model"` // rule-engine-v1 | llm-enhance-v1 | chat-llm-v1
	Disclaimer      string              `json:"disclaimer,omitempty"`
	TurnID          string              `json:"turn_id,omitempty"`
	ClientMessageID *string             `json:"client_message_id,omitempty"`
	RunStatus       string              `json:"run_status,omitempty"`
	RunLeaseUntil   *time.Time          `json:"run_lease_until,omitempty"`
	Degraded        bool                `json:"degraded,omitempty"`
}

// AgentEvidence 证据条目：引用必须来自真实查询结果，不得虚构（§2.1 / D12）。
type AgentEvidence struct {
	SourceType string    `json:"source_type"` // record | reminder | trend | weight | cat_profile | family_scope
	SourceID   string    `json:"source_id"`   // 记录/提醒 ID；趋势类可空
	CatIDs     []string  `json:"cat_ids,omitempty"`
	OccurredAt time.Time `json:"occurred_at,omitempty"`
	Excerpt    string    `json:"excerpt,omitempty"` // 原始文案节选，禁止改写诊断
}

// AgentAction 操作建议：导航类（不可写库）与提醒草稿类（需用户确认）。
type AgentAction struct {
	Type    string         `json:"type"` // view_records | view_trend | create_reminder
	Title   string         `json:"title"`
	CatID   string         `json:"cat_id,omitempty"`  // 单猫导航/提醒必填；家庭级留空
	Days    int            `json:"days,omitempty"`    // 趋势/记录查看窗口
	Payload map[string]any `json:"payload,omitempty"` // 仅导航类允许携带限定键
}

// AgentMessageType 消息类型常量。
const (
	AgentTypePatrolAbnormal = "patrol_abnormal"
	AgentTypePatrolWeight   = "patrol_weight"
	AgentTypeChatAnswer     = "chat_answer"
	AgentTypeReminderDraft  = "reminder_draft"
)

// 提醒草稿状态机：pending → confirmed / dismissed / expired（§2.1）。
const (
	DraftPending   = "pending"
	DraftConfirmed = "confirmed"
	DraftDismissed = "dismissed"
	DraftExpired   = "expired"
)

// AgentRule 规则标识（AG-A03 固定语义）。
const (
	RuleR01 = "R-01" // 昨日 danger 记录
	RuleR02 = "R-02" // 7 天 ≥3 次呕吐/稀便
	RuleR03 = "R-03" // 用药提醒超期 24h
	RuleR04 = "R-04" // 14 天未称重（周日 20:00）
	RuleR05 = "R-05" // 疫苗/驱虫 7 天内到期
	RuleR06 = "R-06" // 昨日家庭零记录
)

// ── 错误码（AG-01 约定，对应 §2 第 5 条）──────────────────

// Agent 专属业务错误码，复用 errors 包常量风格。
// 映射：AGENT_DISABLED/UNAVAILABLE → 503；
// AGENT_INVALID_OUTPUT → 422；AGENT_DRAFT_EXPIRED/CONFLICT → 409；
// 越权/不存在 → 现有 CodeNotFound/CodeFamilyForbidden（不泄露存在性）。
const (
	CodeAgentDisabled     = errors.CodeAgentDisabled
	CodeAgentUnavailable  = errors.CodeAgentUnavailable
	CodeAgentInvalidOut   = errors.CodeAgentInvalidOutput
	CodeAgentDraftExpired = errors.CodeAgentDraftExpired
	CodeAgentConflict     = errors.CodeAgentConflict
)

// ── 请求/响应 DTO（AG-07 路由直接绑定）────────────────────

// AgentMessageListQuery GET /agent/messages 查询参数。
// type/status 不可识别时返回 400，不默认返回全部（§8 合同要求）。
type AgentMessageListQuery struct {
	Type   string
	Status string
	Limit  int
	Before string // 游标：generated_at|id，不跨家庭
}

// AgentMessageListResponse 家庭消息列表（仅 visibility=family 且 status≠dismissed）。
type AgentMessageListResponse struct {
	Messages   []*AgentMessage `json:"messages"`
	NextCursor string          `json:"next_cursor"`
}

// AgentChatRequest POST /agent/chat。
type AgentChatRequest struct {
	SessionID string `json:"session_id"`
	// ClientMessageID 幂等键：同会话内唯一；重试返回原结果而非重发（AG-B04）。
	ClientMessageID string `json:"client_message_id"`
	Message         string `json:"message"`
}

// AgentChatResponse 聊天响应。
type AgentChatResponse struct {
	SessionID       string        `json:"session_id"`
	TurnID          string        `json:"turn_id"`
	ClientMessageID string        `json:"client_message_id"`
	Status          string        `json:"status"`
	Message         *AgentMessage `json:"message"`
	Degraded        bool          `json:"degraded"` // true=模型不可用，返回的是确定性兜底
}

// AgentReminderDraft 可确认的提醒草稿（一条消息至多携带一份，§2.1）。
type AgentReminderDraft struct {
	MessageID string              `json:"message_id"`
	Version   int                 `json:"version"`
	ExpiresAt time.Time           `json:"expires_at"`
	Reminder  *AgentReminderInput `json:"reminder"`
	// 仅 Agent 生成侧填写；确认接口（AG-A05）只接受 expected_version，
	// 草稿本体从消息行内加载，不再信任客户端任意 payload（§2.1 / D04）。
	ConfirmedReminderID string `json:"confirmed_reminder_id,omitempty"`
}

// AgentReminderEditRequest PATCH /agent/messages/:id/draft。
// expected_version：首次保存为 0，保存成功后为 1；旧版本返回 AGENT_CONFLICT。
type AgentReminderEditRequest struct {
	ExpectedVersion int                 `json:"expected_version"`
	Reminder        *AgentReminderInput `json:"reminder"`
}

// AgentReminderInput 是 Agent 草稿专用输入，避免污染旧 reminders API 的 camelCase 字段。
type AgentReminderInput struct {
	CatID       string     `json:"cat_id,omitempty"`
	Type        string     `json:"type"`
	Title       string     `json:"title"`
	Subtitle    string     `json:"subtitle,omitempty"`
	Time        string     `json:"time,omitempty"`
	ScheduledAt *time.Time `json:"scheduled_at,omitempty"`
	Timezone    string     `json:"timezone,omitempty"`
	Icon        string     `json:"icon,omitempty"`
	Rule        string     `json:"rule,omitempty"`
}

// AgentConfirmRequest POST /agent/messages/:id/confirm。
type AgentConfirmRequest struct {
	ExpectedVersion int `json:"expected_version"`
}

// AgentConfirmResponse 确认结果：幂等，重复确认返回同一提醒。
type AgentConfirmResponse struct {
	Reminder     *AgentReminderResult `json:"reminder"`
	MessageID    string               `json:"message_id"`
	ActionStatus string               `json:"action_status"`
}

type AgentReminderResult struct {
	ID          string     `json:"id"`
	FamilyID    string     `json:"family_id"`
	CatID       string     `json:"cat_id"`
	Type        string     `json:"type"`
	Title       string     `json:"title"`
	Subtitle    string     `json:"subtitle,omitempty"`
	Time        string     `json:"time,omitempty"`
	State       string     `json:"state"`
	Icon        string     `json:"icon,omitempty"`
	Rule        string     `json:"rule,omitempty"`
	ScheduledAt *time.Time `json:"scheduled_at,omitempty"`
	Timezone    string     `json:"timezone,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

// AgentPatrolRequest POST /api/v1/internal/agent/patrol（仅目标家庭 admin）。
type AgentPatrolRequest struct {
	FamilyID string `json:"family_id"`
}

// AgentPatrolResponse 巡检执行结果。
type AgentPatrolResponse struct {
	Messages []*AgentMessage `json:"messages"`
}

// AgentSessionListResponse GET /agent/sessions。
type AgentSessionListResponse struct {
	Sessions   []AgentSessionSummary `json:"sessions"`
	NextCursor string                `json:"next_cursor"`
}

// AgentSessionSummary 会话摘要（仅本人私有会话，§8）。
type AgentSessionSummary struct {
	ID        string    `json:"id"`
	Title     string    `json:"title,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// AgentSessionMessagesResponse GET /agent/sessions/:id/messages。
type AgentSessionMessagesResponse struct {
	Messages   []*AgentMessage `json:"messages"`
	NextCursor string          `json:"next_cursor"`
}

// 分页/预算上限（写入 OpenAPI 与配置默认值，AG-01 第 6 条）。
const (
	AgentMessageListMaxLimit     = 50
	AgentMessageListDefaultLimit = 20
	AgentChatMessageMaxLen       = 2000 // 用户单条输入上限（字符）
	AgentChatHistoryTurns        = 5    // 每轮携带的历史轮数（D 见 AG-B03）
	AgentChatTurnBudgetSec       = 25   // 单轮总预算，低于小程序 30s 超时
	AgentChatMaxModelCalls       = 4
	AgentChatMaxToolExecs        = 8
	AgentDraftDefaultTTL         = 24 * time.Hour
)
