# 猫管家 Agent 接口契约（AG-01）

权威类型位于 `backend/internal/app/agent_dto.go` 和 `miniprogram/src/types/agent.ts`，HTTP 路径与 Schema 位于 `backend/openapi/openapi.yaml`。本文件补充三者无法直接表达的业务边界。所有 Agent HTTP 接口位于 `/api/v1` 下，除忽略成功返回 204 外，响应均使用现有 `{code,message,data,request_id?}` Envelope。

## 身份、可见性与状态

家庭巡检消息为 `role=assistant, visibility=family`，仅该家庭当前成员可读取和操作；私有会话消息为 `visibility=private`，仅会话创建者可读取和续聊。系统巡检消息允许 `user_id` 为空，`created_by=system`。跨家庭或无权访问的资源 ID 返回 `NOT_FOUND`（404），家庭本身无成员资格按现有 `FAMILY_FORBIDDEN`（403）处理。对外的消息不包含完整工具原始响应。

一条消息最多有一份提醒草稿。草稿状态为 `pending → confirmed | dismissed | expired`；展示隐藏状态单独保存在 `display_status`，忽略已确认消息不撤销正式提醒，导航动作不更改草稿状态。消息详情可返回已保存的 `draft_reminder` 和 `draft_expires_at` 供编辑恢复。草稿编辑使用 `expected_version`，成功后版本加一；确认只提交版本，服务端从持久化消息取草稿。重复确认返回同一个提醒 ID。单猫草稿必须有有效猫咪 ID；家庭级草稿由用户明确选择 `cat_id=both`，缺省不能解释为家庭级。确认前必须补齐未来的绝对 `scheduled_at`。

## 消息、证据与分页

消息 JSON 使用 snake_case。`session_id`、`cat_id` 对非对应范围可省略；`role`、`visibility`、`generated_at`、`model` 必须存在。证据格式为 `{source_type,source_id,cat_ids?,occurred_at?,excerpt?}`：R-03/R-05 引用提醒 ID；R-04 引用最近称重记录 ID（`source_type=weight`），若从未称重则引用猫咪建档 ID（`source_type=cat_profile`）；R-06 使用家庭 ID 和昨日查询范围。`excerpt` 只节选原始信息，不把缺失记录、展示默认值或模型猜测写成医疗事实。

家庭消息列表和会话消息列表的 `data` 始终为 `{messages,next_cursor}`，会话列表为 `{sessions,next_cursor}`。`next_cursor` 无下一页时为空字符串；游标以消息 `(generated_at,id)` 或会话 `(updated_at,id)` 稳定排序，且始终受家庭和会话权限限制。分页默认 20 条，最大 50 条；显式 `limit=0`、负数、超过 50、无效 `type/status/before` 均返回 400。聊天响应 `data` 为 `{session_id,message,degraded}`，确认响应为 `{reminder,message_id,action_status}`。

## 时间、长度与模型预算

家庭日期窗口按家庭 IANA 时区的本地午夜计算，再转换成 UTC 左闭右开区间。R-01/R-06 用昨日；R-02 用最近七个本地日（含今日截至当前）；R-04 仅周日 20:00；R-03 超期严格大于 24 小时；R-05 从当前时刻起至七个本地日后的相同时刻，右边界不含。没有绝对 `scheduled_at` 的历史提醒不参与到期规则。草稿默认有效期 24 小时。

单条聊天内容去除首尾空白后要求 1–2000 个 Unicode 字符；`client_message_id` 必填，长度 1–128。每轮最多携带五轮历史、四次模型请求、八次工具执行，整体最多 25 秒；这些上限在 Go DTO 中作为当前固定默认值，OpenAPI 也记录了对外限制。查询工具还必须受家庭、猫咪归属和分页上限约束，不得把截断结果称作完整统计。模型输出校验失败返回 `AGENT_INVALID_OUTPUT`（422）或在已有可靠查询结果时明确降级；模型不可用返回 `AGENT_UNAVAILABLE`（503）或明确的 `degraded=true`。降级内容只能基于本轮实际成功查询到的授权数据，不能编造命中数量或来源。

## 开关与错误码

`AI_AGENT_ENABLED` 默认 false，控制巡检与新草稿确认；关闭后历史仍可读、可忽略。`AI_ENABLED` 默认 false，仅控制模型使用；Agent 开启但模型关闭时规则巡检仍执行。`AI_AGENT_LLM_ENHANCE` 默认 false，仅允许普通巡检正文增强，不更改证据、风险等级和动作；danger 正文保留规则模板。

| Code | HTTP | 含义 |
|---|---:|---|
| `AGENT_DISABLED` | 503 | Agent 开关关闭，写操作停止 |
| `AGENT_UNAVAILABLE` | 503 | 模型或必要依赖不可用 |
| `AGENT_INVALID_OUTPUT` | 422 | 模型输出未通过结构或事实校验 |
| `AGENT_DRAFT_EXPIRED` | 409 | 草稿超过有效期或不可确认 |
| `AGENT_CONFLICT` | 409 | 版本、状态或幂等键冲突 |
| `VALIDATION_FAILED` | 400 | 请求参数或内容越界 |
| `RATE_LIMITED` | 429 | 手动巡检频率超限；进程内每来源 IP 每分钟最多 3 次 |
| `FAMILY_FORBIDDEN` / `NOT_FOUND` | 403 / 404 | 家庭权限不足 / 资源不可见 |

阶段 B/C 的模型能力、工具调用和正文增强需要对应任务的实现与验收；本契约本身不代表这些能力已交付。
