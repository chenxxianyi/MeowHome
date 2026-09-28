// Agent（猫管家）契约类型 — 与 backend/internal/app/agent_dto.go 逐字段对应（AG-01 冻结契约）。
// 列表响应固定为 { messages, next_cursor }；聊天响应固定为 { session_id, message, degraded }；
// 确认响应固定为 { reminder, message_id, action_status }；外层仍走现有 Envelope 解包。
import type { ID } from './index'

export type AgentRole = 'user' | 'assistant' | 'tool'
export type AgentVisibility = 'family' | 'private'
export type AgentMessageType =
  | 'patrol_abnormal'
  | 'patrol_weight'
  | 'chat_answer'
  | 'reminder_draft'
export type AgentActionStatus = 'pending' | 'confirmed' | 'dismissed' | 'expired'
export type AgentModel = 'rule-engine-v1' | 'llm-enhance-v1' | 'chat-llm-v1'

export interface AgentEvidence {
  source_type: 'record' | 'reminder' | 'trend' | 'weight' | 'family_scope'
  source_id: string
  cat_ids?: ID[]
  occurred_at?: string // RFC3339
  excerpt?: string // 原始文案节选，前端不得改写
}

export interface AgentActionSuggestion {
  type: 'view_records' | 'view_trend' | 'create_reminder'
  title: string
  cat_id?: ID | 'both'
  days?: number
  payload?: Record<string, unknown>
}

export interface AgentMessage {
  id: ID
  session_id?: ID
  role: AgentRole
  visibility: AgentVisibility
  family_id?: ID
  user_id?: ID
  cat_id?: ID
  type: AgentMessageType
  severity: 'info' | 'warning' | 'danger'
  title: string
  body: string
  evidence?: AgentEvidence[]
  action_suggestions?: AgentActionSuggestion[]
  draft_version?: number
  action_status?: AgentActionStatus
  generated_at: string // RFC3339
  model: AgentModel
  disclaimer?: string
}

export interface AgentReminderDraft {
  message_id: ID
  version: number
  expires_at: string
  reminder?: AgentReminderInput
  confirmed_reminder_id?: ID
}

// ── 请求体 ──

export interface AgentMessageListParams {
  type?: AgentMessageType
  status?: AgentActionStatus
  limit?: number
  before?: string // 游标 generated_at|id
}

export interface AgentChatRequest {
  session_id?: ID
  client_message_id: string // 幂等键，同会话内唯一
  message: string
}

export interface AgentReminderEditRequest {
  expected_version: number
  reminder: AgentReminderInput
}

export interface AgentConfirmRequest {
  expected_version: number
}

// ── 响应体 ──

export interface AgentMessageListResponse {
  messages: AgentMessage[]
  next_cursor?: string
}

export interface AgentChatResponse {
  session_id: ID
  message: AgentMessage
  degraded: boolean
}

export interface AgentConfirmResponse {
  reminder: AgentReminderResult
  message_id: ID
  action_status: AgentActionStatus
}

export interface AgentReminderResult {
  id: ID
  family_id: ID
  cat_id: ID | 'both'
  type: string
  title: string
  subtitle?: string
  time?: string
  state: string
  icon?: string
  rule?: string
  scheduled_at?: string
  timezone?: string
  completed_at?: string
}

export interface AgentReminderInput {
  cat_id?: ID | 'both'
  type:
    | 'medication'
    | 'vaccine'
    | 'deworm'
    | 'checkup'
    | 'weight'
    | 'inventory'
    | 'water'
    | 'custom'
  title: string
  subtitle?: string
  time?: string // 展示用 time_label 兼容
  scheduled_at?: string // RFC3339，可计算的绝对时间（AG-A02）
  timezone?: string // IANA 时区，展示用
  icon?: string
  rule?: string
}

export interface AgentPatrolResponse {
  messages: AgentMessage[]
}

export interface AgentSessionSummary {
  id: ID
  title?: string
  created_at: string
  updated_at: string
}

export interface AgentSessionListResponse {
  sessions: AgentSessionSummary[]
  next_cursor?: string
}

export interface AgentSessionMessagesResponse {
  messages: AgentMessage[]
  next_cursor?: string
}

// ── 错误码（与后端对齐；request() 解包后按 code 分支）──

export type AgentErrorCode =
  | 'AGENT_DISABLED'
  | 'AGENT_UNAVAILABLE'
  | 'AGENT_INVALID_OUTPUT'
  | 'AGENT_DRAFT_EXPIRED'
  | 'AGENT_CONFLICT'
  | 'FAMILY_FORBIDDEN'
  | 'NOT_FOUND'
  | 'VALIDATION_FAILED'
