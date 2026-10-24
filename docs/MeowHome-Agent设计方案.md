# MeowHome 猫管家 Agent 设计方案

> 文档版本：v1.0
> 编制日期：2026-09-25
> 前置依赖：`MeowHome-开发方案.md` 4.3 节（AI 与 OCR 规范）；后端 `internal/app/ai_service.go` 现有 `Parse` / `Summary` 能力

---

## 1. 定位与目标

### 1.1 定位

**猫管家（MeowHome Agent）** 是在现有 AI 摘要（`/ai/summary`）和自然语言解析（`/ai/parse`）之上，增加一层**主动式、对话式**的智能体。

当前系统 AI 能力边界：

| 能力 | 触发方式 | 说明 |
|------|----------|------|
| 自然语言解析 | 用户手动输入 → `ParseAI` | 用户驱动，被动 |
| 每日摘要 | 用户打开今日页 → `AISummary` | 被动，每次手动触发 |
| **Agent（本方案新增）** | **规则/事件自动触发 + 用户对话** | **主动，双向** |

Agent 的三个核心能力：

1. **主动巡检**：基于记录数据自动识别异常，生成带依据的健康提示，无需用户提问。
2. **智能问答**：用户用自然语言问"小白最近吃得怎么样"，Agent 查询数据后回答。
3. **行动建议**：Agent 可以建议创建提醒（如"建议明天 8:00 复查尿检"），生成待确认的提醒草稿，由用户确认后才入库——**绝不绕过用户直接写库**。

### 1.2 与现有 AI 功能的关系

```
现有链路：  用户输入 → AIService.Parse() → 结构化记录（草稿）→ 用户确认 → 入库
现有链路：  今日页   → AIService.Summary() → 摘要文本 → 展示

新增链路：  AgentRunner（定时/事件）→ AgentPlan() → 规则 + LLM → AgentMessage（带证据）
                                          ↘
新增链路：  用户对话 → AgentChat() → ToolCalling LLM → 工具结果 → 最终回答
```

Agent **复用**现有 `AIService` 的解析逻辑，不重新实现；新增的是"主动触发"和"多轮对话 + 工具调用"两个维度。

---

## 2. 安全边界（与开发方案 §2.3 对齐）

以下约束对 Agent 的所有输出强制执行，**不可由 LLM 覆盖**：

| 约束 | 实现方式 |
|------|----------|
| Agent 不能直接写健康记录 | Agent 输出仅包含"建议"，通过 `createReminderDraft` 工具生成草稿，必须用户确认 |
| Agent 不能访问任意 SQL | 工具层（Tool）白名单：`listRecords`、`getCatProfile`、`getTrends`、`createReminderDraft`，参数校验由后端完成 |
| 医疗结论必须带免责声明 | Agent 输出模板末尾强制附加"本分析仅供参考，不构成医疗诊断" |
| 数据隔离 | 所有工具调用携带 `family_id` + `user_id`，Agent 服务层先做 `requireFamilyAccess`，与 `AIService.Summary` 一致 |
| 可降级 | `AI_ENABLED=false` 时 Agent 主动巡检停止（摘要降级为确定性统计），用户对话端展示"AI 管家暂未启用" |

---

## 3. 功能设计

### 3.1 主动巡检（Agent Patrol）

**触发时机**（后台定时任务）：

| 频率 | 触发条件 | 动作 |
|------|----------|------|
| 每日 08:00 | 昨日有 danger/warning 记录 | 生成"昨日异常回顾"摘要 |
| 每日 08:00 | 有未完成用药提醒超期 | 生成"用药超期提醒" |
| 每周日 20:00 | 有猫咪超过 14 天未记录体重 | 生成"体重监测提醒" |
| 事件驱动 | 新记录 severity=danger | 立即生成"紧急提示"推送 |

**输出格式**（`AgentMessage`）：

```json
{
  "id": "am_xxx",
  "family_id": "fam_xxx",
  "type": "patrol_abnormal",
  "severity": "warning",
  "title": "小白的呕吐记录需要关注",
  "body": "昨天 14:30 记录了呕吐（黄色液体），且 7 天内已有 2 次类似记录。建议观察今天进食情况，如持续建议就诊。",
  "evidence": [
    {"record_id": "rec_xxx", "cat_name": "小白", "time": "昨天 14:30", "type": "vomit", "note": "黄色液体"}
  ],
  "action_suggestions": [
    {"type": "create_reminder", "title": "明天早上复查呕吐情况", "time": "明天 08:00"},
    {"type": "view_trend", "title": "查看小白 7 天健康趋势"}
  ],
  "generated_at": "2026-09-25T08:00:00Z",
  "model": "rule-engine-v1",
  "disclaimer": "本分析仅供参考，不构成医疗诊断。"
}
```

**第一阶段实现**（`rule-engine-v1`）：纯确定性规则，不依赖 LLM，保证可降级：

```go
// PatrolRules 主动巡检规则列表（顺序执行，任一命中即生成 AgentMessage）
type PatrolRules []Rule

// Rule 单条巡检规则
type Rule struct {
  Name     string        // 规则名，用于审计
  Match    func(ctx context.Context, familyCtx FamilyContext) bool
  Message  func(ctx context.Context, familyCtx FamilyContext) *AgentMessage
}

// FamilyContext 巡检时预加载的家庭上下文
type FamilyContext struct {
  FamilyID  string
  UserID    string
  Cats      []model.Cat
  Records   []model.DailyRecord   // 近 7 天
  Reminders []model.Reminder      // 待办
  Trends    []TrendPoint
}
```

**第二阶段扩展**（`llm-enhance`）：规则命中后，将 `FamilyContext` 摘要 + 命中记录交给 LLM 生成自然语言正文，替换 `body` 字段；**evidence 和 action_suggestions 仍由规则引擎生成**，LLM 只负责"措辞"，不影响结论准确性。

### 3.2 智能问答（Agent Chat）

**交互方式**：小程序新增"猫管家"入口（悬浮按钮或"今日页"顶部），点击后进入对话页。

**工具集（Tool Calling）**：

| 工具名 | 入参 | 返回 | 说明 |
|--------|------|------|------|
| `listRecords` | `cat_id`, `type?`, `days?`, `limit?` | `RecordDTO[]` | 查询近 N 天记录，按类型过滤 |
| `getCatProfile` | `cat_id` | `CatDTO`（含健康档案） | 获取猫咪基础档案 |
| `getTrends` | `cat_id`, `metric`, `days` | `TrendPoint[]` | 获取指定指标趋势 |
| `createReminderDraft` | `title`, `time`, `cat_id?` | `ReminderDraft` | 生成待确认提醒草稿，**不入库** |

**Agent 对话流程**：

```
用户："小白这两天吃得怎么样？"
  ↓
LLM → 工具调用 listRecords(cat_id=cat_xiaobai, type="feed", days=2)
  ↓
工具返回 4 条记录
  ↓
LLM → 组织回答："小白今天记录了 2 次喂食（早上 7:00、中午 12:00），
       昨天记录 2 次。总体来看喂食频率正常，但今天没有喝水记录。"
  ↓
AgentMessage（type="chat_answer"，带 evidence 记录 ID）
```

**上下文管理**：每轮对话携带近 5 条对话历史（token 截断）；`family_id` 和 `user_id` 由会话绑定，不暴露给 LLM 自由填充，防止越权。

**降级策略**：LLM 不可用时，对话端返回确定性回答："目前 AI 管家暂未启用，已为您查询到小白近 2 天共 4 条喂食记录：…"（直接展示工具结果，不经过 LLM 组织）。

### 3.3 行动建议（Agent Actions）

Agent 可以建议两类操作，均以"草稿"形式呈现，**用户确认后才生效**：

**① 创建提醒草稿**

触发示例：Agent 发现"小白连续 3 天有呕吐记录" → 建议创建"明天带小白就医"提醒。

前端 UI（`ai-confirm` 页面扩展）：

```
┌─────────────────────────────────────┐
│ 🐱 猫管家建议                      │
│                                     │
│  检测到小白近 3 天有 2 次呕吐记录， │
│  建议明天上午带小白去宠物医院复查。 │
│                                     │
│  ┌ 提醒草稿 ─────────────────────┐  │
│  │ 明天 10:00  带小白去宠物医院  │  │
│  │ 关联猫咪：小白               │  │
│  └───────────────────────────────┘  │
│                                     │
│  [编辑时间/内容]  [确认创建]  [忽略] │
└─────────────────────────────────────┘
```

确认后调用 `POST /families/:id/reminders`（已有端点），Agent 消息标记 `action_status = "confirmed"`。

**② 导航引导**

Agent 消息可携带 `action_suggestions` 中的 `view_trend` 类型，前端识别后渲染为"查看趋势"按钮，点击直接跳转 `/records?cat=xxx&days=7`。

---

## 4. 数据模型

### 4.1 新增表

**`ai_agent_messages`**（Agent 消息/草稿）

```sql
CREATE TABLE ai_agent_messages (
  id             VARCHAR(36) PRIMARY KEY,
  family_id      VARCHAR(36) NOT NULL,
  user_id        VARCHAR(36) NOT NULL,
  type           VARCHAR(32) NOT NULL,      -- patrol_abnormal | patrol_weight | chat_answer | reminder_draft
  severity       VARCHAR(16) NOT NULL DEFAULT 'info',
  title          VARCHAR(200) NOT NULL,
  body           TEXT NOT NULL,
  evidence_json  TEXT,                       -- AIEvidence[] JSON
  action_json    TEXT,                       -- ActionSuggestion[] JSON
  action_status  VARCHAR(16) NOT NULL DEFAULT 'pending',  -- pending | confirmed | dismissed
  model          VARCHAR(32) NOT NULL,       -- rule-engine-v1 | llm-enhance | chat-llm-v1
  generated_at   DATETIME NOT NULL,
  created_at     DATETIME NOT NULL,
  updated_at     DATETIME NOT NULL,
  deleted_at     DATETIME,
  INDEX idx_family_type (family_id, type, generated_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

**`ai_agent_sessions`**（对话会话）

```sql
CREATE TABLE ai_agent_sessions (
  id             VARCHAR(36) PRIMARY KEY,
  family_id      VARCHAR(36) NOT NULL,
  user_id        VARCHAR(36) NOT NULL,
  title          VARCHAR(200),               -- 首条消息摘要
  status         VARCHAR(16) NOT NULL DEFAULT 'active',  -- active | closed
  created_at     DATETIME NOT NULL,
  updated_at     DATETIME NOT NULL,
  INDEX idx_family_user (family_id, user_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

**`ai_agent_messages` 复用现有 `ai_sessions` 表**（`AIService.Parse` 已在使用），对话消息单独存 `ai_agent_messages`，避免与解析会话混淆。

### 4.2 新增 GORM 模型

```go
// internal/domain/model/agent.go

type AgentMessage struct {
  Base
  FamilyID     string `gorm:"size:36;not null"`
  UserID       string `gorm:"size:36;not null"`
  Type         string `gorm:"size:32;not null"`
  Severity     string `gorm:"size:16;not null;default:'info'"`
  Title        string `gorm:"size:200;not null"`
  Body         string `gorm:"type:text"`
  EvidenceJSON string `gorm:"type:text"`
  ActionJSON   string `gorm:"type:text"`
  ActionStatus string `gorm:"size:16;not null;default:'pending'"`
  Model        string `gorm:"size:32;not null"`
  GeneratedAt  time.Time
}

type AgentSession struct {
  Base
  FamilyID string `gorm:"size:36;not null"`
  UserID   string `gorm:"size:36;not null"`
  Title    string `gorm:"size:200"`
  Status   string `gorm:"size:16;not null;default:'active'"`
}
```

---

## 5. 后端架构

### 5.1 新增文件

```
internal/app/agent_service.go        # Agent 业务逻辑（巡检 + 问答 + 行动）
internal/app/agent_rules.go          # 确定性规则引擎（第一阶段）
internal/app/agent_tools.go          # Tool 定义（供 LLM 调用）
internal/domain/model/agent.go       # GORM 模型
internal/domain/repository/agent_repo.go   # Repository 接口
internal/infrastructure/persistence/mysql/agent_repo.go
internal/transport/http/handler/agent.go   # HTTP handler
```

### 5.2 AgentService 接口设计

```go
type AgentService struct {
  messages  repository.AgentMessageRepo
  sessions  repository.AgentSessionRepo
  records   repository.DailyRecordRepo
  cats      repository.CatRepo
  reminders repository.ReminderRepo
  care      *CareService          // 复用趋势数据
  ai        *AIService            // 复用解析能力
  llm       LLMProvider          // 新增，可选
}

// ── 主动巡检 ─────────────────────────────────────────────────

// Patrol 对指定家庭执行全部巡检规则，生成 AgentMessage 列表。
// 由定时任务调用，结果落库并推送。
func (s *AgentService) Patrol(ctx context.Context, familyID, userID string) ([]*AgentMessage, error)

// ── 智能问答 ─────────────────────────────────────────────────

// Chat 处理用户自然语言提问，LLM 工具调用，返回 AgentMessage。
// 降级：LLM 不可用时直接返回工具查询结果。
func (s *AgentService) Chat(ctx context.Context, familyID, userID, sessionID, userMessage string) (*AgentMessage, error)

// ── 行动建议 ─────────────────────────────────────────────────

// ConfirmReminderDraft 用户确认后，将草稿转为真实提醒，写入 reminders 表。
func (s *AgentService) ConfirmReminderDraft(ctx context.Context, familyID, userID, messageID string) (*model.Reminder, error)

// DismissMessage 用户忽略，标记 action_status = "dismissed"。
func (s *AgentService) DismissMessage(ctx context.Context, familyID, userID, messageID string) error
```

### 5.3 LLM Provider 接口

```go
// LLMProvider 屏蔽具体 LLM 供应商
type LLMProvider interface {
  // ChatWithTools 多轮对话 + 工具调用
  // tools 为 AgentService 注册的工具列表；LLM 自行决定调用哪个工具
  ChatWithTools(ctx context.Context, req LLMRequest, tools []ToolDef) (*LLMResponse, error)
}

type LLMRequest struct {
  Model       string
  Messages    []LLMMessage   // 对话历史（含 system prompt）
  Temperature float64
  MaxTokens   int
}

type LLMMessage struct {
  Role    string   // system | user | assistant | tool
  Content string
  Name    string   // tool 调用结果时携带
}

type ToolDef struct {
  Name        string
  Description string
  Parameters  json.RawMessage  // JSON Schema
}

type LLMResponse struct {
  Content      string
  ToolCalls    []ToolCall   // LLM 请求调用的工具
  Usage        LLMUsage
  Model        string
}
```

**实现**：先用 OpenAI 兼容接口（`AI_BASE_URL` 已配置，默认关闭），`AI_AGENT_ENABLED` 单独控制 Agent 功能（与现有 `AI_ENABLED` 解耦，Agent 未启用时 `AI_ENABLED` 仍可开启摘要功能）。

### 5.4 新增 HTTP 端点

```
# 主动巡检（后台任务调用，需管理员权限）
POST /internal/agent/patrol          body: { family_id }
# 响应：{ messages: AgentMessage[] }

# 对话（用户驱动）
POST /families/:familyId/agent/chat
  body: { session_id?, message }
  响应：{ session_id, message: AgentMessage }

# 获取家庭 Agent 消息列表（今日页展示巡检结果）
GET  /families/:familyId/agent/messages
  query: type?&limit?&before?
  响应：{ messages: AgentMessage[] }

# 确认提醒草稿
POST /families/:familyId/agent/messages/:messageId/confirm
  响应：{ reminder: ReminderDTO }

# 忽略消息
POST /families/:familyId/agent/messages/:messageId/dismiss
  响应：204
```

### 5.5 定时任务

Agent 巡检由后台定时任务触发（复用现有 `JOB` 接口）：

```go
// internal/app/jobs.go（已有或新建）
type AgentPatrolJob struct {
  agent  *AgentService
  families repository.FamilyRepo
}

// Run 每日 08:00 和 20:00 各执行一次
func (j *AgentPatrolJob) Run(ctx context.Context) error {
  families, _ := j.families.ListAll(ctx)
  for _, f := range families {
    _, _ = j.agent.Patrol(ctx, f.ID, f.OwnerID)  // 忽略单个家庭失败
  }
  return nil
}
```

第一阶段不引入 Redis 调度，用 **Go 内置 `time.Ticker`** + 启动时扫描，部署在单个后端进程即可（MVP 阶段家庭数有限）。

---

## 6. 小程序端设计

### 6.1 新增页面

**`src/pages/agent/index.vue`**（猫管家对话页）

```
┌─────────────────────────────────┐
│ ← 猫管家              20:17    │
├─────────────────────────────────┤
│                                 │
│  [AI 管家 08:01]               │
│  小白昨日有 1 条呕吐记录      │
│  （黄色液体），建议今天观察   │
│  进食情况。                    │
│  [查看记录] [创建提醒] [忽略] │
│                                 │
│  [用户 08:02]                  │
│  小白最近吃得怎么样？          │
│                                 │
│  [AI 管家 08:02]               │
│  小白近 2 天共 4 次喂食，     │
│  今天 7:00 和 12:00 各一次，  │
│  频率正常。没有喝水记录。     │
│  [查看趋势] [创建提醒]        │
│                                 │
├─────────────────────────────────┤
│ [请输入…            ] [发送]   │
└─────────────────────────────────┘
```

**`src/pages/agent/index.vue`** 核心逻辑：

```ts
import { agentApi } from '../../api/endpoints'

async function sendMessage(text: string) {
  if (!text.trim()) return
  const localId = `local-${Date.now()}`
  chatMessages.value.push({ id: localId, role: 'user', content: text, at: now() })
  pendingMessageId.value = localId
  try {
    const res = await agentApi.chat(familyId, {
      session_id: sessionId.value,
      message: text
    })
    sessionId.value = res.session_id
    chatMessages.value.push({
      id: res.message.id,
      role: 'assistant',
      content: res.message.body,
      evidence: res.message.evidence,
      actions: res.message.action_suggestions,
      at: res.message.generated_at
    })
  } catch {
    chatMessages.value.push({
      id: localId, role: 'assistant',
      content: 'AI 管家暂时不可用，请稍后重试。'
    })
  }
}
```

**`src/api/endpoints.ts`** 新增：

```ts
export const agentApi = {
  chat(familyId: string, data: { session_id?: string; message: string }) {
    return request<AgentChatResponse>(http, {
      method: 'POST',
      url: `/families/${familyId}/agent/chat`,
      data
    })
  },
  listMessages(familyId: string, params?: { type?: string; limit?: number }) {
    return request<AgentMessage[]>(http, {
      url: `/families/${familyId}/agent/messages`,
      params
    })
  },
  confirm(familyId: string, messageId: string) {
    return request<ReminderDTO>(http, {
      method: 'POST',
      url: `/families/${familyId}/agent/messages/${messageId}/confirm`
    })
  },
  dismiss(familyId: string, messageId: string) {
    return request<void>(http, {
      method: 'POST',
      url: `/families/${familyId}/agent/messages/${messageId}/dismiss`
    })
  }
}
```

### 6.2 今日页展示巡检结果

`/today` 页面顶部（现有 `aiSummary` 区域上方）新增"管家提醒"区块：

```vue
<view v-if="agentAlerts.length" class="today-agent-alerts">
  <view v-for="msg in agentAlerts.slice(0, 2)" :key="msg.id"
        :class="['agent-alert-item', { danger: msg.severity === 'danger' }]">
    <AppIcon :name="msg.severity === 'danger' ? 'warning' : 'info'" :size="16" />
    <text>{{ msg.title }}</text>
    <button @click="router.push('/agent')">详情</button>
  </view>
</view>
```

数据加载：`agentApi.listMessages(familyId, { type: 'patrol_abnormal', limit: 5 })`，仅展示 `action_status = 'pending'` 的消息。

### 6.3 新增文件清单（小程序）

```
src/pages/agent/index.vue
src/api/endpoints.ts          # 追加 agentApi
src/stores/agent.ts           # Agent 会话状态（可选，或直接在页面内管理）
src/styles/pages.css         # 追加 agent-alert-item、agent-chat-* 样式
src/components/agent/AgentEvidenceCard.vue  # 证据卡片组件
src/components/agent/AgentActionBar.vue     # 确认/忽略按钮组
```

---

## 7. 规则引擎详细设计（第一阶段，无 LLM）

### 7.1 规则列表

| 规则 ID | 条件 | 严重度 | 消息标题 |
|---------|------|--------|----------|
| `R-01` | 昨日 danger 记录数 ≥ 1 | danger | `{猫名}有健康异常需要关注` |
| `R-02` | 7 天内同类型记录 ≥ 3（呕吐/腹泻） | warning | `{猫名}近期多次{类型}，建议留意` |
| `R-03` | 用药提醒完成时间超期 > 24h | warning | `用药提醒已超期，请补录` |
| `R-04` | 猫咪 14 天未记录体重 | warning | `{猫名}已超过 14 天未记录体重` |
| `R-05` | 疫苗/驱虫提醒距到期 < 7 天 | info | `{猫名}的{项目}将在{N}天后到期` |
| `R-06` | 昨日 zero records（全家庭） | info | `昨天没有记录，今天记得更新哦` |

### 7.2 规则执行代码

```go
// agent_rules.go

var patrolRules = PatrolRules{
  {
    Name: "R-01",
    Match: func(_ context.Context, fc FamilyContext) bool {
      return fc.CountYesterdayBySeverity("danger") >= 1
    },
    Message: func(_ context.Context, fc FamilyContext) *AgentMessage {
      cats := fc.YesterdayDangerCats()
      return &AgentMessage{
        Type:     "patrol_abnormal",
        Severity: "danger",
        Title:    fmt.Sprintf("%s有健康异常需要关注", cats.Name),
        Body:     buildBody(cats),
        Evidence: fc.YesterdayDangerRecords(),
        Actions: []ActionSuggestion{
          {Type: "view_records", Title: "查看相关记录"},
          {Type: "create_reminder", Title: "创建就医提醒"},
        },
        Model: "rule-engine-v1",
      }
    },
  },
  // R-02 ~ R-06 类似...
}

// RunPatrol 执行全部规则
func RunPatrol(ctx context.Context, fc FamilyContext) []*AgentMessage {
  var results []*AgentMessage
  for _, rule := range patrolRules {
    if rule.Match(ctx, fc) {
      results = append(results, rule.Message(ctx, fc))
    }
  }
  return results
}
```

---

## 8. 实施阶段

### 阶段 A（2 周）：主动巡检 + 消息展示

| 任务 | 文件 | 说明 |
|------|------|------|
| 数据模型 + 迁移 | `model/agent.go`、`cmd/migrate` | 建 2 张表 |
| 规则引擎 | `agent_rules.go` | R-01 ~ R-06 确定性规则 |
| 后台巡检任务 | `agent_service.go` + `jobs.go` | `Patrol()` + `time.Ticker` |
| Agent HTTP 端点 | `handler/agent.go`、`router.go` | 4 个端点 |
| 小程序 agentApi | `endpoints.ts` | 4 个方法 |
| 今日页巡检展示 | `today/index.vue` | `agentAlerts` 区块 |
| 确认/忽略操作 | `today/index.vue` 或独立操作 | 调用 confirm/dismiss |

**验收标准**：家庭昨日有 danger 记录 → 今日页顶部展示"小白有健康异常需要关注"→ 点击详情可跳转 Agent 对话页 → 用户确认后提醒写入 reminders 表。

### 阶段 B（2 周）：对话问答

| 任务 | 文件 | 说明 |
|------|------|------|
| LLMProvider 接口 + 实现 | `agent_tools.go`、`llm/openai.go` | OpenAI 兼容，tool calling |
| 4 个 Agent Tool | `agent_tools.go` | listRecords / getCatProfile / getTrends / createReminderDraft |
| AgentChat 端点 | `handler/agent.go` | 对话逻辑 |
| Agent 对话页 | `pages/agent/index.vue` | 完整对话 UI |
| 降级处理 | 对话页 catch | LLM 不可用时展示工具结果 |

**验收标准**：用户输入"小白最近吃得怎么样" → Agent 调用 `listRecords` → 返回自然语言回答 + 证据卡片 → 用户可点"创建提醒"确认。

### 阶段 C（1 周）：LLM 增强巡检正文

| 任务 | 文件 | 说明 |
|------|------|------|
| LLM 增强巡检 | `agent_service.go` `Patrol()` | 规则命中后 LLM 生成 body，evidence/actions 不变 |
| 灰度开关 | `agent_service.go` | `AI_AGENT_LLM_ENHANCE` 配置项 |

---

## 9. 配置项（追加到 `.env.example`）

```ini
# --- Agent（猫管家）---
AI_AGENT_ENABLED=false          # 独立开关，false 时巡检停止、对话降级
AI_AGENT_PATROL_TIMES=08:00,20:00
AI_AGENT_LLM_ENHANCE=false      # 规则命中后用 LLM 改写 body
```

---

## 10. 与现有代码的集成点

| 现有位置 | 集成方式 |
|----------|----------|
| `ai_service.go` `AIService` | Agent 复用 `AIService` 的 `requireFamilyAccess` 和 records/cats 仓储，不重新实现 |
| `care_service.go` `CareService.Trends` | Agent 工具 `getTrends` 直接调用 |
| `reminder_service.go` | `ConfirmReminderDraft` 调用现有 reminder 创建逻辑 |
| `router.go` | 新增 `familyG` 下 4 个 Agent 路由，与现有 `/ai/parse`、`/ai/summary` 并列 |
| `AI_ENABLED` 配置 | Agent 使用独立的 `AI_AGENT_ENABLED`，两者解耦 |
| `ai_sessions` 表 | 对话会话使用新表 `ai_agent_sessions`，不复用 |

---

## 11. 测试策略

| 层级 | 范围 | 工具 |
|------|------|------|
| 规则引擎 | 每条规则 R-01~R-06 的命中/不命中边界 | Go 单元测试 |
| AgentService | Patrol / Chat / ConfirmReminderDraft 主流程 | Go 集成测试（Mock LLM） |
| LLM Provider | Tool calling 格式正确性 | 接口 Mock，不依赖真实 LLM |
| 权限隔离 | 跨家庭 Agent 调用被拒绝 | 集成测试 |
| 小程序 | agentApi 4 个端点、降级展示 | 手动 DevTools 验证 |
| 确认提醒 | 确认后 reminders 表有记录 | 集成测试 |

**不做的事**：Agent 输出质量（措辞准确性）不纳入自动化测试，上线前人工审阅 LLM 样本。

---

## 12. 风险与缓解

| 风险 | 影响 | 缓解 |
|------|------|------|
| LLM 产生幻觉，给出错误医疗建议 | 用户误信 | evidence 强制来自工具查询，body 免责声明；高风险规则（R-01）直接跳 LLM，规则引擎生成 |
| Tool calling 参数注入（跨家庭） | 越权访问 | `family_id` 由后端会话注入，不从 LLM 输出读取；工具层做 `requireFamilyAccess` |
| 巡检任务阻塞主服务 | 服务卡顿 | 巡检任务在独立 goroutine，超时 30s 单家庭；失败跳过，不影响其他家庭 |
| LLM 供应商不可用 | 对话降级 | 已有降级路径；巡检第一阶段（规则引擎）完全不依赖 LLM |
| 提醒草稿堆积 | 用户看到大量未确认消息 | 每日巡检上限 3 条（`limit` 参数）；dismiss 后不再展示 |

---

## 13. 下一步

- [ ] 确认阶段 A 开始时间，建 `cmd/migrate` 新增 2 张表
- [ ] 确认 LLM 供应商（OpenAI / 国产模型），`AI_BASE_URL` 填入实际地址
- [ ] 确认是否需要"紧急提示"（R-01）推送通知（微信模板消息），或仅应用内展示
- [ ] 评审本方案后同步更新 `MeowHome-开发方案.md` §4.3 和 §5（项目结构）
